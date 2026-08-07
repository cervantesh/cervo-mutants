package report

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/cervantesh/cervo-mutants/pkg/engine"
)

type GovernanceReview struct {
	SchemaVersion          string                          `json:"schema_version"`
	GeneratedAt            string                          `json:"generated_at"`
	QuarantinePolicy       GovernanceQuarantinePolicy      `json:"quarantine_policy"`
	ActiveQuarantineCount  int                             `json:"active_quarantine_count"`
	ExpiredQuarantineCount int                             `json:"expired_quarantine_count"`
	QuarantineTemplates    []GovernanceQuarantineTemplate  `json:"quarantine_templates,omitempty"`
	SuppressionTemplates   []GovernanceSuppressionTemplate `json:"suppression_templates,omitempty"`
}

type GovernanceQuarantinePolicy struct {
	Path          string `json:"path,omitempty"`
	ExpireAfter   string `json:"expire_after,omitempty"`
	RequireReason bool   `json:"require_reason,omitempty"`
	RequireOwner  bool   `json:"require_owner,omitempty"`
	RequireIssue  bool   `json:"require_issue,omitempty"`
	FailOnExpired bool   `json:"fail_on_expired,omitempty"`
	MaxRenewals   int    `json:"max_renewals,omitempty"`
}

type GovernanceQuarantineTemplate struct {
	MutantID        string                    `json:"mutant_id"`
	Status          engine.Status             `json:"status"`
	SuggestedAction string                    `json:"suggested_action"`
	Template        GovernanceQuarantineEntry `json:"template"`
	Guidance        []string                  `json:"guidance,omitempty"`
	SuggestedJSON   string                    `json:"suggested_json,omitempty"`
}

type GovernanceQuarantineEntry struct {
	MutantID  string `json:"mutant_id"`
	Reason    string `json:"reason"`
	Owner     string `json:"owner"`
	Issue     string `json:"issue"`
	CreatedAt string `json:"created_at,omitempty"`
	ExpiresAt string `json:"expires_at,omitempty"`
	Renewals  int    `json:"renewals"`
}

type GovernanceSuppressionTemplate struct {
	MutantID        string                    `json:"mutant_id"`
	Status          engine.Status             `json:"status"`
	SuggestedAction string                    `json:"suggested_action"`
	Rule            GovernanceSuppressionRule `json:"rule"`
	Guidance        []string                  `json:"guidance,omitempty"`
	SuggestedYAML   string                    `json:"suggested_yaml,omitempty"`
}

type GovernanceSuppressionRule struct {
	Name           string `json:"name"`
	Operator       string `json:"operator,omitempty"`
	File           string `json:"file,omitempty"`
	Original       string `json:"original,omitempty"`
	Mutated        string `json:"mutated,omitempty"`
	EquivalentRisk string `json:"equivalent_risk,omitempty"`
	Action         string `json:"action"`
	Reason         string `json:"reason"`
	Evidence       string `json:"evidence,omitempty"`
	Reviewers      int    `json:"reviewers,omitempty"`
}

func GovernanceReviewJSON(result engine.RunResult) ([]byte, error) {
	return governanceReviewJSONAt(result, time.Now().UTC())
}

func governanceReviewJSONAt(result engine.RunResult, now time.Time) ([]byte, error) {
	review := buildGovernanceReview(result, now)
	return json.MarshalIndent(review, "", "  ")
}

func GovernanceReviewMarkdown(result engine.RunResult) string {
	return governanceReviewMarkdownAt(result, time.Now().UTC())
}

func governanceReviewMarkdownAt(result engine.RunResult, now time.Time) string {
	review := buildGovernanceReview(result, now)
	var b strings.Builder
	b.WriteString("# CervoMutants Governance Review\n\n")
	fmt.Fprintf(&b, "- Active quarantine entries: **%d**\n", review.ActiveQuarantineCount)
	fmt.Fprintf(&b, "- Expired quarantine entries: **%d**\n", review.ExpiredQuarantineCount)
	fmt.Fprintf(&b, "- Suggested quarantine templates: **%d**\n", len(review.QuarantineTemplates))
	fmt.Fprintf(&b, "- Suggested suppression templates: **%d**\n", len(review.SuppressionTemplates))
	if review.QuarantinePolicy.Path != "" {
		fmt.Fprintf(&b, "- Quarantine path: `%s`\n", review.QuarantinePolicy.Path)
	}
	if review.QuarantinePolicy.ExpireAfter != "" {
		fmt.Fprintf(&b, "- Default quarantine expiry window: `%s`\n", review.QuarantinePolicy.ExpireAfter)
	}
	fmt.Fprintf(&b, "- Quarantine requires: reason=`%t` owner=`%t` issue=`%t`\n", review.QuarantinePolicy.RequireReason, review.QuarantinePolicy.RequireOwner, review.QuarantinePolicy.RequireIssue)
	fmt.Fprintf(&b, "- Fail on expired quarantine: `%t`\n", review.QuarantinePolicy.FailOnExpired)
	fmt.Fprintf(&b, "- Max renewals: `%d`\n\n", review.QuarantinePolicy.MaxRenewals)

	b.WriteString("## Quarantine Templates\n\n")
	if len(review.QuarantineTemplates) == 0 {
		b.WriteString("No quarantine templates were suggested.\n\n")
	} else {
		for _, item := range review.QuarantineTemplates {
			fmt.Fprintf(&b, "### `%s`\n\n", item.MutantID)
			fmt.Fprintf(&b, "- Status: `%s`\n", item.Status)
			fmt.Fprintf(&b, "- Suggested action: `%s`\n", item.SuggestedAction)
			for _, guidance := range item.Guidance {
				fmt.Fprintf(&b, "- Guidance: %s\n", guidance)
			}
			if item.SuggestedJSON != "" {
				b.WriteString("\n```json\n")
				b.WriteString(item.SuggestedJSON)
				b.WriteString("\n```\n\n")
			}
		}
	}

	b.WriteString("## Suppression Templates\n\n")
	if len(review.SuppressionTemplates) == 0 {
		b.WriteString("No suppression templates were suggested.\n")
	} else {
		for _, item := range review.SuppressionTemplates {
			fmt.Fprintf(&b, "### `%s`\n\n", item.MutantID)
			fmt.Fprintf(&b, "- Status: `%s`\n", item.Status)
			fmt.Fprintf(&b, "- Suggested action: `%s`\n", item.SuggestedAction)
			for _, guidance := range item.Guidance {
				fmt.Fprintf(&b, "- Guidance: %s\n", guidance)
			}
			if item.SuggestedYAML != "" {
				b.WriteString("\n```yaml\n")
				b.WriteString(item.SuggestedYAML)
				b.WriteString("\n```\n\n")
			}
		}
	}
	return b.String()
}

func buildGovernanceReview(result engine.RunResult, now time.Time) GovernanceReview {
	review := GovernanceReview{
		SchemaVersion:          "1",
		GeneratedAt:            now.Format(time.RFC3339),
		ActiveQuarantineCount:  result.Quarantine.Active,
		ExpiredQuarantineCount: result.Quarantine.Expired,
		QuarantinePolicy: GovernanceQuarantinePolicy{
			Path:          result.Quarantine.Path,
			ExpireAfter:   result.Quarantine.ExpireAfter,
			RequireReason: result.Quarantine.RequireReason,
			RequireOwner:  result.Quarantine.RequireOwner,
			RequireIssue:  result.Quarantine.RequireIssue,
			FailOnExpired: result.Quarantine.FailOnExpired,
			MaxRenewals:   result.Quarantine.MaxRenewals,
		},
		QuarantineTemplates:  buildQuarantineTemplates(result, now),
		SuppressionTemplates: buildSuppressionTemplates(result),
	}
	return review
}

func buildQuarantineTemplates(result engine.RunResult, now time.Time) []GovernanceQuarantineTemplate {
	items := make([]GovernanceQuarantineTemplate, 0)
	seen := map[string]bool{}
	for _, mutant := range result.Mutants {
		if mutant.Status == engine.StatusQuarantined {
			continue
		}
		if seen[mutant.MutantID] {
			continue
		}
		suggestedAction, reason, ok := quarantineTemplateSeed(mutant)
		if !ok {
			continue
		}
		entry := GovernanceQuarantineEntry{
			MutantID:  mutant.MutantID,
			Reason:    reason,
			Owner:     ownershipRouteReviewOwner(mutant.Mutant.Ownership),
			Issue:     "",
			CreatedAt: now.Format(time.RFC3339),
			ExpiresAt: governanceSuggestedExpiry(result.Quarantine, now),
			Renewals:  0,
		}
		data, _ := json.MarshalIndent(entry, "", "  ")
		items = append(items, GovernanceQuarantineTemplate{
			MutantID:        mutant.MutantID,
			Status:          mutant.Status,
			SuggestedAction: suggestedAction,
			Template:        entry,
			Guidance:        quarantineGuidance(result.Quarantine, mutant),
			SuggestedJSON:   string(data),
		})
		seen[mutant.MutantID] = true
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].MutantID < items[j].MutantID
	})
	return items
}

func quarantineTemplateSeed(mutant engine.MutantResult) (suggestedAction, reason string, ok bool) {
	if mutant.Status == engine.StatusTimedOut && mutant.FailureKind == "non_progress_loop" {
		return "quarantine", governanceReason(mutant, "reviewed-skip or quarantine if timeout confirms a non-progress loop"), true
	}
	audit, ok := governanceStrongestAudit(mutant.Mutant.SuppressionAudit)
	if !ok || audit.Action != "quarantine-required" {
		return "", "", false
	}
	return "quarantine", governanceReason(mutant, audit.Reason), true
}

func quarantineGuidance(policy engine.QuarantineStats, mutant engine.MutantResult) []string {
	guidance := []string{
		fmt.Sprintf("policy.fail_on_expired=%t", policy.FailOnExpired),
		fmt.Sprintf("policy.max_renewals=%d", policy.MaxRenewals),
	}
	if policy.RequireOwner {
		guidance = append(guidance, "owner is required before this template can be activated")
	}
	if policy.RequireIssue {
		guidance = append(guidance, "issue is required before this template can be activated")
	}
	if policy.RequireReason {
		guidance = append(guidance, "reason must stay specific and audit-ready")
	}
	if policy.ExpireAfter != "" {
		guidance = append(guidance, "default expiry window="+policy.ExpireAfter)
	}
	if mutant.FailureKind == "non_progress_loop" {
		guidance = append(guidance, "confirm the timeout is reproducibly non-progress before activating quarantine")
	}
	if ownership := ownershipRouteSummary(mutant.Mutant.Ownership); ownership != "" {
		guidance = append(guidance, "ownership_route="+ownership)
	}
	return guidance
}

func governanceSuggestedExpiry(policy engine.QuarantineStats, now time.Time) string {
	if strings.TrimSpace(policy.ExpireAfter) == "" {
		return ""
	}
	duration, err := time.ParseDuration(policy.ExpireAfter)
	if err != nil {
		return ""
	}
	return now.Add(duration).UTC().Format(time.RFC3339)
}

func buildSuppressionTemplates(result engine.RunResult) []GovernanceSuppressionTemplate {
	items := make([]GovernanceSuppressionTemplate, 0)
	seen := map[string]bool{}
	for _, mutant := range result.Mutants {
		if seen[mutant.MutantID] {
			continue
		}
		audit, ok := governanceStrongestAudit(mutant.Mutant.SuppressionAudit)
		if !ok {
			continue
		}
		rule := GovernanceSuppressionRule{
			Name:           governanceRuleName(mutant, audit),
			Operator:       mutant.Mutant.Operator,
			File:           filepath.ToSlash(mutant.Mutant.File),
			Original:       mutant.Mutant.Original,
			Mutated:        mutant.Mutant.Mutated,
			EquivalentRisk: mutant.Mutant.EquivalentRisk,
			Action:         audit.Action,
			Reason:         governanceReason(mutant, audit.Reason),
			Evidence:       fallbackText(audit.EvidenceLevel, "heuristic"),
			Reviewers:      audit.ReviewerCount,
		}
		items = append(items, GovernanceSuppressionTemplate{
			MutantID:        mutant.MutantID,
			Status:          mutant.Status,
			SuggestedAction: audit.Action,
			Rule:            rule,
			Guidance:        suppressionGuidance(mutant, audit),
			SuggestedYAML:   governanceSuppressionYAML(rule),
		})
		seen[mutant.MutantID] = true
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].MutantID < items[j].MutantID
	})
	return items
}

func governanceStrongestAudit(audits []engine.SuppressionAudit) (engine.SuppressionAudit, bool) {
	bestPriority := -1
	var best engine.SuppressionAudit
	for _, audit := range audits {
		priority := governanceSuppressionPriority(audit.Action)
		if priority > bestPriority {
			best = audit
			bestPriority = priority
		}
	}
	return best, bestPriority >= 0
}

func governanceSuppressionPriority(action string) int {
	switch action {
	case "report-only":
		return 0
	case "lower-priority":
		return 1
	case "quarantine-required":
		return 2
	case "suppress":
		return 3
	default:
		return -1
	}
}

func governanceRuleName(mutant engine.MutantResult, audit engine.SuppressionAudit) string {
	if strings.TrimSpace(audit.Name) != "" {
		return audit.Name
	}
	base := strings.TrimSuffix(filepath.Base(mutant.Mutant.File), filepath.Ext(mutant.Mutant.File))
	return "review-" + base + "-" + mutant.Mutant.Operator + "-" + strconv.Itoa(mutant.Mutant.Line)
}

func governanceReason(mutant engine.MutantResult, fallback string) string {
	for _, value := range []string{mutant.SuggestedSkipReason, fallback, mutant.StatusReason} {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return "review this mutant before creating a durable rule"
}

func suppressionGuidance(mutant engine.MutantResult, audit engine.SuppressionAudit) []string {
	guidance := []string{
		"evidence=" + fallbackText(audit.EvidenceLevel, "heuristic"),
		fmt.Sprintf("reviewers=%d", audit.ReviewerCount),
	}
	if audit.Action == "suppress" {
		guidance = append(guidance, "confirmed evidence and at least one reviewer are required before suppressing")
	}
	if mutant.Mutant.EquivalentRisk != "" {
		guidance = append(guidance, "equivalent_risk="+mutant.Mutant.EquivalentRisk)
	}
	return guidance
}

func governanceSuppressionYAML(rule GovernanceSuppressionRule) string {
	var b strings.Builder
	fmt.Fprintf(&b, "- name: %s\n", rule.Name)
	if rule.Operator != "" {
		fmt.Fprintf(&b, "  operator: %s\n", rule.Operator)
	}
	if rule.File != "" {
		fmt.Fprintf(&b, "  file: %s\n", rule.File)
	}
	if rule.Original != "" {
		fmt.Fprintf(&b, "  original: %s\n", quoteYAML(rule.Original))
	}
	if rule.Mutated != "" {
		fmt.Fprintf(&b, "  mutated: %s\n", quoteYAML(rule.Mutated))
	}
	if rule.EquivalentRisk != "" {
		fmt.Fprintf(&b, "  equivalent_risk: %s\n", rule.EquivalentRisk)
	}
	fmt.Fprintf(&b, "  action: %s\n", rule.Action)
	fmt.Fprintf(&b, "  reason: %s\n", quoteYAML(rule.Reason))
	if rule.Evidence != "" {
		fmt.Fprintf(&b, "  evidence: %s\n", rule.Evidence)
	}
	if rule.Reviewers > 0 {
		fmt.Fprintf(&b, "  reviewers: %d\n", rule.Reviewers)
	}
	return strings.TrimRight(b.String(), "\n")
}

func quoteYAML(value string) string {
	escaped := strings.ReplaceAll(value, `"`, `\"`)
	return `"` + escaped + `"`
}
