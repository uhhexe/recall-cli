// Copyright 2026 ühh and contributors. Licensed under MIT. See LICENSE.
//
// Hand-written. Recall.ai does not expose a single "give me the transcript
// for this bot" endpoint: a bot has recordings, a recording has a transcript
// artifact (once one was requested and has finished processing), and the
// transcript artifact carries a download_url for the actual word-level JSON
// (see https://docs.recall.ai/docs/download-schemas). This command chains
// those three steps behind one call so the CLI's transcript UX matches how
// people actually talk about the feature: "get me the transcript for bot X."
//
// The bot fetch (step 1) goes through the generated client, so it is subject
// to --dry-run and --region exactly like every other command. The transcript
// download (step 2) fetches a pre-signed URL that is not part of the Recall
// API surface and needs no Recall auth header, so it uses a small direct
// HTTP call instead of internal/client.Client.
package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/spf13/cobra"

	"recall/internal/cliutil"
)

// botTranscriptView is the shape 'recall bots transcript' prints. It is a
// hand-assembled result, not a raw API passthrough, so --json marshals this
// struct directly rather than relaying upstream bytes.
type botTranscriptView struct {
	BotID            string                   `json:"bot_id"`
	RecordingID      string                   `json:"recording_id,omitempty"`
	TranscriptStatus string                   `json:"transcript_status"`
	DownloadURL      string                   `json:"download_url,omitempty"`
	Utterances       []botTranscriptUtterance `json:"utterances,omitempty"`
	RawTranscript    json.RawMessage          `json:"raw_transcript,omitempty"`
	Note             string                   `json:"note,omitempty"`
}

type botTranscriptUtterance struct {
	Speaker string `json:"speaker,omitempty"`
	Text    string `json:"text"`
}

// recallBotForTranscript covers only the fields this command reads off the
// bot object. The full Bot shape has many more (see 'recall bots get
// --help'); keeping this local and narrow means a field Recall adds upstream
// can't silently break JSON decoding here.
type recallBotForTranscript struct {
	ID         string `json:"id"`
	Recordings []struct {
		ID             string `json:"id"`
		MediaShortcuts *struct {
			Transcript *struct {
				ID     string `json:"id"`
				Status struct {
					Code string `json:"code"`
				} `json:"status"`
				Data struct {
					DownloadURL string `json:"download_url"`
				} `json:"data"`
			} `json:"transcript"`
		} `json:"media_shortcuts"`
	} `json:"recordings"`
}

func newBotsTranscriptCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transcript <id>",
		Short: "Fetch the transcript for a bot's most recent recording, if one is ready.",
		Long: `Fetch the transcript for a bot's most recent recording, if one is ready.

This is a convenience command, not a single Recall.ai endpoint: it retrieves
the bot, finds its most recent recording, and follows that recording's
transcript artifact to the download URL (GET /api/v1/bot/{id}/ then a plain
fetch of the pre-signed transcript URL Recall returns). A bot only has a
transcript if it was created with a transcription provider configured, for
example 'recall bots create --meeting-url <url> --transcribe'.`,
		Example: strings.Trim(`
  recall bots transcript 550e8400-e29b-41d4-a716-446655440000
  recall bots transcript 550e8400-e29b-41d4-a716-446655440000 --json
  recall bots transcript 550e8400-e29b-41d4-a716-446655440000 --dry-run
`, "\n"),
		Annotations: map[string]string{
			"pp:endpoint":   "bots.transcript",
			"pp:method":     "GET",
			"pp:path":       "/api/v1/bot/{id}/",
			"mcp:read-only": "true",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			if args[0] == "" {
				return usageErr(fmt.Errorf("id is required\nUsage: %s <id>", cmd.CommandPath()))
			}
			botID := args[0]

			c, err := flags.newClient()
			if err != nil {
				return err
			}

			path := replacePathParam("/api/v1/bot/{id}/", "id", botID)
			data, err := c.Get(cmd.Context(), path, nil)
			if err != nil {
				return classifyAPIError(err, flags)
			}
			if dryRunOK(flags) {
				fmt.Fprintln(cmd.ErrOrStderr(), "would then look for a finished transcript on this bot's most recent recording and, if found, fetch its download URL")
				return flags.printJSON(cmd, map[string]any{"dry_run": true})
			}

			var bot recallBotForTranscript
			if err := json.Unmarshal(data, &bot); err != nil {
				return fmt.Errorf("parsing bot response: %w", err)
			}

			view := botTranscriptView{BotID: botID}
			if len(bot.Recordings) == 0 {
				view.TranscriptStatus = "no_recording"
				view.Note = "This bot has no recordings yet. It may still be in the call, or it may not have joined."
				return printBotTranscriptView(cmd, flags, view)
			}

			// Most recent recording first: Recall returns recordings in
			// creation order, so the last entry is the most recent one.
			rec := bot.Recordings[len(bot.Recordings)-1]
			view.RecordingID = rec.ID

			if rec.MediaShortcuts == nil || rec.MediaShortcuts.Transcript == nil {
				view.TranscriptStatus = "not_configured"
				view.Note = "This recording has no transcript artifact. Create the bot with --transcribe to request one."
				return printBotTranscriptView(cmd, flags, view)
			}

			transcript := rec.MediaShortcuts.Transcript
			view.TranscriptStatus = transcript.Status.Code
			if transcript.Status.Code != "done" {
				view.Note = "Transcript is not ready yet (status: " + transcript.Status.Code + "). Recall processes transcripts after the call ends; try again shortly."
				return printBotTranscriptView(cmd, flags, view)
			}
			if transcript.Data.DownloadURL == "" {
				view.Note = "Transcript is marked done but Recall did not return a download URL."
				return printBotTranscriptView(cmd, flags, view)
			}
			view.DownloadURL = transcript.Data.DownloadURL

			body, err := fetchTranscriptDownload(cmd, flags, view.DownloadURL)
			if err != nil {
				view.Note = "Transcript is done, but fetching the download URL failed: " + err.Error()
				return printBotTranscriptView(cmd, flags, view)
			}
			applyTranscriptBody(&view, body)
			return printBotTranscriptView(cmd, flags, view)
		},
	}
	return cmd
}

// fetchTranscriptDownload fetches Recall's pre-signed transcript URL
// directly. It intentionally does not go through internal/client.Client:
// the download URL is not a Recall API path (no base_url join, no
// Authorization header), and it lives on a different host per call.
func fetchTranscriptDownload(cmd *cobra.Command, flags *rootFlags, url string) ([]byte, error) {
	ctx, cancel := boundCtx(cmd.Context(), flags)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	httpClient := &http.Client{}
	if cliutil.IsDogfoodEnv() || cliutil.IsVerifyEnv() {
		// Defense in depth: a spec-derived mock server or dogfood fixture
		// should never hand back a real download_url, but if one somehow
		// does, don't let a hand-written command hang past the matrix's
		// per-command budget.
		httpClient.Timeout = 5_000_000_000 // 5s, avoids importing "time" for one literal
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20)) // 10MB cap; transcripts are text-only
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("download URL returned HTTP %d", resp.StatusCode)
	}
	return body, nil
}

// recallTranscriptDownloadEntry mirrors the per-participant shape documented
// at https://docs.recall.ai/docs/download-schemas for the transcript
// download URL: one entry per speaker turn, each carrying that speaker's
// words. This has not been verified against a live download (no API key was
// available while building this CLI) — see README Known Gaps. Any shape
// mismatch falls back to raw_transcript so no data is silently dropped.
type recallTranscriptDownloadEntry struct {
	Participant *struct {
		Name string `json:"name"`
	} `json:"participant"`
	Words []struct {
		Text string `json:"text"`
	} `json:"words"`
}

// applyTranscriptBody tries to decode body as the documented per-speaker
// download shape. On any mismatch it keeps the raw bytes under
// raw_transcript instead of guessing, so --json output is always either a
// clean speaker/text breakdown or the untouched upstream payload, never a
// half-parsed hybrid.
func applyTranscriptBody(view *botTranscriptView, body []byte) {
	var entries []recallTranscriptDownloadEntry
	hasWords := false
	if err := json.Unmarshal(body, &entries); err == nil && len(entries) > 0 {
		for _, e := range entries {
			if len(e.Words) > 0 {
				hasWords = true
				break
			}
		}
	}
	if !hasWords {
		view.RawTranscript = json.RawMessage(body)
		view.Note = "Downloaded the transcript, but its shape didn't match the documented per-speaker format; showing the raw payload in raw_transcript."
		return
	}
	for _, e := range entries {
		if len(e.Words) == 0 {
			continue
		}
		words := make([]string, 0, len(e.Words))
		for _, w := range e.Words {
			words = append(words, w.Text)
		}
		speaker := "unknown speaker"
		if e.Participant != nil && e.Participant.Name != "" {
			speaker = e.Participant.Name
		}
		view.Utterances = append(view.Utterances, botTranscriptUtterance{
			Speaker: speaker,
			Text:    strings.Join(words, " "),
		})
	}
}

func printBotTranscriptView(cmd *cobra.Command, flags *rootFlags, view botTranscriptView) error {
	if flags.asJSON || !wantsHumanTable(cmd.OutOrStdout(), flags) {
		return flags.printJSON(cmd, view)
	}
	w := cmd.OutOrStdout()
	fmt.Fprintf(w, "bot:    %s\n", view.BotID)
	if view.RecordingID != "" {
		fmt.Fprintf(w, "recording: %s\n", view.RecordingID)
	}
	fmt.Fprintf(w, "status: %s\n", view.TranscriptStatus)
	if len(view.Utterances) > 0 {
		fmt.Fprintln(w)
		for _, u := range view.Utterances {
			if u.Speaker != "" {
				fmt.Fprintf(w, "%s: %s\n", u.Speaker, u.Text)
			} else {
				fmt.Fprintln(w, u.Text)
			}
		}
	} else if len(view.RawTranscript) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, string(view.RawTranscript))
	}
	if view.Note != "" {
		fmt.Fprintf(w, "\nnote: %s\n", view.Note)
	}
	return nil
}
