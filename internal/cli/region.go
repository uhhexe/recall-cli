// Copyright 2026 ühh and contributors. Licensed under MIT. See LICENSE.

package cli

import (
	"fmt"
	"os"
	"strings"

	"recall/internal/config"
)

// recallRegions lists every Recall.ai region documented at
// https://docs.recall.ai/docs/regions as of 2026-07-26. Each region is a
// fully separate deployment: bots, recordings, and transcripts created in
// one region are not visible from another, and API keys are region-scoped.
var recallRegions = map[string]bool{
	"us-east-1":      true,
	"us-west-2":      true,
	"eu-central-1":   true,
	"ap-northeast-1": true,
}

// defaultRecallRegion matches the spec's default base_url and Recall's own
// "if you're calling api.recall.ai, that's us-east-1" documented default.
const defaultRecallRegion = "us-east-1"

// applyRegion resolves --region / $RECALL_REGION into cfg.BaseURL.
//
// Precedence, highest first:
//  1. $RECALL_BASE_URL — an explicit full base URL override. Always wins.
//     This is also how 'verify' points the CLI at its spec-derived mock
//     server without a real Recall.ai account, so region resolution must
//     never clobber it.
//  2. --region flag
//  3. $RECALL_REGION environment variable
//  4. the spec default (us-east-1), already set by config.Load.
func applyRegion(cfg *config.Config, regionFlag string) error {
	if strings.TrimSpace(os.Getenv("RECALL_BASE_URL")) != "" {
		return nil
	}
	region := strings.TrimSpace(regionFlag)
	source := "--region"
	if region == "" {
		region = strings.TrimSpace(os.Getenv("RECALL_REGION"))
		source = "$RECALL_REGION"
	}
	if region == "" {
		return nil
	}
	if !recallRegions[region] {
		return fmt.Errorf(
			"unknown %s value %q: must be one of us-east-1, us-west-2, eu-central-1, ap-northeast-1. "+
				"Each Recall.ai region is a separate deployment with its own bots, recordings, and API key; "+
				"see https://docs.recall.ai/docs/regions. For a non-standard endpoint (e.g. a local mock "+
				"server), set RECALL_BASE_URL directly instead",
			source, region,
		)
	}
	cfg.BaseURL = fmt.Sprintf("https://%s.recall.ai", region)
	return nil
}

// resolvedRegion reports the region that applyRegion would pick, purely for
// display in 'recall auth status' and 'recall doctor' — it never mutates
// config.
func resolvedRegion(regionFlag string) string {
	if v := strings.TrimSpace(os.Getenv("RECALL_BASE_URL")); v != "" {
		return "custom (RECALL_BASE_URL=" + v + ")"
	}
	if region := strings.TrimSpace(regionFlag); region != "" {
		return region
	}
	if region := strings.TrimSpace(os.Getenv("RECALL_REGION")); region != "" {
		return region
	}
	return defaultRecallRegion + " (default)"
}
