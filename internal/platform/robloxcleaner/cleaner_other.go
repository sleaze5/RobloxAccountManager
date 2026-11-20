//go:build !windows

package robloxcleaner

import "context"

const Supported = false

type cleanupTarget struct{}

func listPhaseTargets(Phase) []cleanupTarget { return nil }

func deleteTargets(context.Context, []cleanupTarget, *progressTracker) {}

func skipReasonOf(error) SkipReason { return SkipReasonFailed }
