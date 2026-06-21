package report

import (
	"fmt"
	"html"
	"sort"
	"strings"
	"time"

	"github.com/cervantesh/cervo-mutants/pkg/engine"
)

type htmlReportRow struct {
	MutantID                  string
	Status                    string
	Survivor                  bool
	SurvivorRank              int
	Actionability             string
	Operator                  string
	EquivalentRisk            string
	GroupFilter               string
	GroupLabel                string
	HistoryStatus             string
	AgeBand                   string
	AgeLabel                  string
	TimingBand                string
	TimingLabel               string
	DurationText              string
	File                      string
	Line                      int
	Function                  string
	Original                  string
	Mutated                   string
	Description               string
	FailureKind               string
	StatusReason              string
	SuggestedSkip             string
	SuggestedScope            string
	NearestTests              string
	RecommendationSummary     string
	RecommendationStrategy    string
	RecommendationPrimaryTest string
	RecommendationAssertions  string
	RankReason                string
	Diff                      string
	Actionable                bool
	PlatformSensitive         bool
	NonProgressRisk           string
	Owner                     string
	Team                      string
	OwnershipSummary          string
	Search                    string
}

type htmlFilterOption struct {
	Value string
	Label string
	Count int
}

type htmlFilterModel struct {
	Status        []htmlFilterOption
	Actionability []htmlFilterOption
	Operator      []htmlFilterOption
	Risk          []htmlFilterOption
	Group         []htmlFilterOption
	Owner         []htmlFilterOption
	Team          []htmlFilterOption
	History       []htmlFilterOption
	Age           []htmlFilterOption
	Timing        []htmlFilterOption
}

type htmlSummaryModel struct {
	ActionableSurvivors int
	SurvivorGroups      int
	LongStanding        int
	SlowSignals         int
}

type htmlPageModel struct {
	Result            engine.RunResult
	Rows              []htmlReportRow
	HasOwnership      bool
	Filters           htmlFilterModel
	Summary           htmlSummaryModel
	InitialVisible    int
	InitialGroupCount int
	RawSummary        string
}

const htmlWorkbenchStyles = `<style>
body{margin:0;font-family:Segoe UI,Arial,sans-serif;background:#f5f7fb;color:#162033}
.page{max-width:1600px;margin:0 auto;padding:24px}
.hero{display:grid;gap:16px;margin-bottom:20px}
.hero h1{margin:0;font-size:30px}
.hero p{margin:0;color:#44506a;max-width:960px}
.cards{display:grid;grid-template-columns:repeat(auto-fit,minmax(180px,1fr));gap:12px}
.card{background:#fff;border:1px solid #d9e1f2;border-radius:14px;padding:14px 16px;box-shadow:0 4px 18px rgba(22,32,51,.05)}
.card-label{display:block;font-size:12px;font-weight:700;letter-spacing:.04em;text-transform:uppercase;color:#61708e}
.card-value{display:block;margin-top:8px;font-size:28px;font-weight:700}
.toolbar,.table-shell,.summary-shell{background:#fff;border:1px solid #d9e1f2;border-radius:14px;box-shadow:0 4px 18px rgba(22,32,51,.05)}
.toolbar{padding:16px;margin-bottom:20px}
.toolbar h2,.table-shell h2{margin:0 0 12px 0;font-size:18px}
.toolbar-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(170px,1fr));gap:12px;align-items:end}
.toolbar label{display:flex;flex-direction:column;gap:6px;font-size:13px;font-weight:600;color:#34415c}
.toolbar input,.toolbar select,.toolbar button{font:inherit}
.toolbar input,.toolbar select{padding:9px 10px;border:1px solid #bec9df;border-radius:10px;background:#fff}
.toolbar button{padding:10px 12px;border:0;border-radius:10px;background:#163b73;color:#fff;font-weight:700;cursor:pointer}
.toolbar button.quick-filter{background:#eaf0fb;color:#163b73}
.toolbar button:hover{filter:brightness(.98)}
.checkbox-label{justify-content:flex-end}
.checkbox-label input{margin-right:8px}
.quick-nav{display:grid;gap:10px;margin-top:16px}
.chip-row{display:flex;flex-wrap:wrap;gap:8px}
.chip-row strong{display:inline-flex;align-items:center;font-size:13px;color:#44506a}
.results-meta{margin-top:16px;font-size:13px;color:#44506a}
.table-shell{padding:16px}
table{width:100%;border-collapse:collapse}
th,td{padding:12px 10px;border-top:1px solid #e2e8f4;vertical-align:top;text-align:left}
thead th{border-top:0;font-size:12px;text-transform:uppercase;letter-spacing:.04em;color:#61708e}
tbody tr:hover{background:#f8fafe}
.badge{display:inline-flex;align-items:center;gap:6px;padding:4px 8px;border-radius:999px;font-size:12px;font-weight:700;background:#edf2ff;color:#27457a}
.badge-survived{background:#fce8cc;color:#8a5100}
.badge-killed{background:#dff7e5;color:#176338}
.badge-timed_out,.badge-memory_killed,.badge-compile_error{background:#fde2e1;color:#8d261d}
.badge-not_covered,.badge-pending_budget,.badge-skipped_resource,.badge-cached,.badge-quarantined,.badge-ignored,.badge-skipped{background:#eceff5;color:#55637d}
.mutant-id{font-family:Consolas,monospace;font-size:12px;color:#49556e}
.mutant-main{display:grid;gap:4px}
.mutant-file{font-weight:700}
.mutant-meta{font-size:12px;color:#61708e}
.mutant-swap{font-family:Consolas,monospace;font-size:12px;color:#24314a}
.mutant-reason{font-size:12px;color:#44506a;max-width:360px}
.diff-shell details{min-width:320px}
.diff-shell summary{cursor:pointer;font-weight:600;color:#163b73}
.diff-shell pre{margin:8px 0 0 0;padding:10px;border-radius:10px;background:#0f1727;color:#e5eefc;overflow:auto;font-size:12px}
.summary-shell{margin-top:20px;padding:16px}
.summary-shell details summary{cursor:pointer;font-weight:700;color:#163b73}
.summary-shell pre{white-space:pre-wrap}
@media (max-width:960px){.page{padding:16px}.table-shell{overflow:auto}}
</style>`

func HTML(result engine.RunResult) string {
	rows := buildHTMLRows(result)
	filters := buildHTMLFilterModel(rows)
	page := buildHTMLPageModel(result, rows, filters)
	return renderHTMLPage(page)
}

func buildHTMLFilterModel(rows []htmlReportRow) htmlFilterModel {
	return htmlFilterModel{
		Status: htmlFilterOptions(rows, func(row htmlReportRow) (string, string) {
			return row.Status, strings.ReplaceAll(row.Status, "_", " ")
		}),
		Actionability: htmlFilterOptions(rows, func(row htmlReportRow) (string, string) {
			return row.Actionability, row.Actionability
		}),
		Operator: htmlFilterOptions(rows, func(row htmlReportRow) (string, string) {
			return row.Operator, row.Operator
		}),
		Risk: htmlFilterOptions(rows, func(row htmlReportRow) (string, string) {
			return row.EquivalentRisk, row.EquivalentRisk
		}),
		Group: htmlFilterOptions(rows, func(row htmlReportRow) (string, string) {
			return row.GroupFilter, row.GroupLabel
		}),
		Owner: htmlFilterOptions(rows, func(row htmlReportRow) (string, string) {
			return row.Owner, row.Owner
		}),
		Team: htmlFilterOptions(rows, func(row htmlReportRow) (string, string) {
			return row.Team, row.Team
		}),
		History: htmlFilterOptions(rows, func(row htmlReportRow) (string, string) {
			return row.HistoryStatus, strings.ReplaceAll(row.HistoryStatus, "_", " ")
		}),
		Age: htmlFilterOptions(rows, func(row htmlReportRow) (string, string) {
			return row.AgeBand, row.AgeLabel
		}),
		Timing: htmlFilterOptions(rows, func(row htmlReportRow) (string, string) {
			return row.TimingBand, row.TimingLabel
		}),
	}
}

func buildHTMLPageModel(result engine.RunResult, rows []htmlReportRow, filters htmlFilterModel) htmlPageModel {
	summary := buildHTMLSummaryModel(rows)
	if result.Summary.Actionable.TrueActionableSurvivors > 0 || result.Summary.Actionable.ActionableSurvivors > 0 {
		summary.ActionableSurvivors = result.Summary.Actionable.TrueActionableSurvivors
	}
	if result.Summary.Actionable.SemanticGroupReviewUnits > 0 {
		summary.SurvivorGroups = result.Summary.Actionable.SemanticGroupReviewUnits
	}
	initialVisible, initialGroups := initialHTMLVisibility(rows)
	return htmlPageModel{
		Result:            result,
		Rows:              rows,
		HasOwnership:      hasOwnershipRoutes(result.Mutants),
		Filters:           filters,
		Summary:           summary,
		InitialVisible:    initialVisible,
		InitialGroupCount: initialGroups,
		RawSummary:        Summary(result),
	}
}

func renderHTMLPage(page htmlPageModel) string {
	var b strings.Builder
	renderHTMLHead(&b)
	renderHTMLHero(&b, page)
	renderHTMLToolbar(&b, page)
	renderHTMLTable(&b, page)
	renderHTMLSummary(&b, page)
	renderHTMLScript(&b, page)
	b.WriteString("</div></body></html>")
	return b.String()
}

func renderHTMLHead(b *strings.Builder) {
	b.WriteString("<!doctype html><html><head><meta charset=\"utf-8\">")
	b.WriteString("<title>cervomut survivor review workbench</title>")
	b.WriteString(htmlWorkbenchStyles)
	b.WriteString("</head><body><div class=\"page\">")
}

func renderHTMLHero(b *strings.Builder, page htmlPageModel) {
	b.WriteString("<section class=\"hero\"><div>")
	b.WriteString("<h1>cervomut survivor review workbench</h1>")
	b.WriteString("<p>Survivors stay first-class, but the full raw run remains in the table below. Use the filters to narrow review by actionability, semantic grouping, operator, equivalent risk, survivor history, age, and timing signal without mutating the underlying report.</p>")
	b.WriteString("</div><div class=\"cards\">")
	writeHTMLCard(b, "Survivors", page.Result.Summary.Survived)
	writeHTMLCard(b, "Actionable score", fmt.Sprintf("%.2f%%", page.Result.Summary.Actionable.ActionableScore))
	writeHTMLCard(b, "True actionable survivors", page.Summary.ActionableSurvivors)
	writeHTMLCard(b, "Semantic review units", page.Summary.SurvivorGroups)
	writeHTMLCard(b, "Long-standing survivors", page.Summary.LongStanding)
	writeHTMLCard(b, "Slow timing signals", page.Summary.SlowSignals)
	writeHTMLCard(b, "Raw score", fmt.Sprintf("%.2f%%", page.Result.Summary.Score))
	b.WriteString("</div></section>")
}

func renderHTMLToolbar(b *strings.Builder, page htmlPageModel) {
	b.WriteString("<section class=\"toolbar\"><h2>Filters</h2><div class=\"toolbar-grid\">")
	writeHTMLInput(b, "filter-search", "Search", "mutant id, file, operator, tests, reason")
	writeHTMLSelect(b, "filter-status", "Status", page.Filters.Status, "All statuses")
	writeHTMLSelect(b, "filter-actionability", "Actionability", page.Filters.Actionability, "All actionability")
	writeHTMLSelect(b, "filter-operator", "Operator", page.Filters.Operator, "All operators")
	writeHTMLSelect(b, "filter-risk", "Equivalent risk", page.Filters.Risk, "All risk levels")
	writeHTMLSelect(b, "filter-group", "Semantic group", page.Filters.Group, "All groups")
	if page.HasOwnership {
		writeHTMLSelect(b, "filter-owner", "Owner", page.Filters.Owner, "All owners")
		writeHTMLSelect(b, "filter-team", "Team", page.Filters.Team, "All teams")
	}
	writeHTMLSelect(b, "filter-history", "History", page.Filters.History, "All history")
	writeHTMLSelect(b, "filter-age", "Survivor age", page.Filters.Age, "All age bands")
	writeHTMLSelect(b, "filter-timing", "Timing signal", page.Filters.Timing, "All timing signals")
	b.WriteString(`<label class="checkbox-label"><span>Primary queue</span><span><input id="filter-survivors-only" type="checkbox" checked>Survivors only</span></label>`)
	b.WriteString(`<label><span>Reset</span><button id="filter-reset" type="button">Reset filters</button></label>`)
	b.WriteString("</div>")
	writeHTMLQuickFilters(b, "Group shortcuts", "filter-group", topHTMLFilterOptions(page.Filters.Group, 6))
	writeHTMLQuickFilters(b, "Operator shortcuts", "filter-operator", topHTMLFilterOptions(page.Filters.Operator, 6))
	fmt.Fprintf(b, `<div class="results-meta">Showing <strong id="visible-count">%d</strong> of <strong id="total-count">%d</strong> mutants. Visible survivors: <strong id="visible-survivors">%d</strong>. Visible semantic groups: <strong id="visible-groups">%d</strong>.</div>`, page.InitialVisible, len(page.Rows), page.InitialVisible, page.InitialGroupCount)
	b.WriteString("</section>")
}

func renderHTMLTable(b *strings.Builder, page htmlPageModel) {
	b.WriteString(`<section class="table-shell"><h2>Review queue</h2><table id="mutant-table"><thead><tr><th>Rank</th><th>Mutant</th><th>Status</th><th>Review signal</th><th>History and timing</th><th>Reason and skip guidance</th><th>Diff</th></tr></thead><tbody>`)
	for _, row := range page.Rows {
		renderHTMLRow(b, row, page.HasOwnership)
	}
	b.WriteString("</tbody></table></section>")
}

func renderHTMLRow(b *strings.Builder, row htmlReportRow, hasOwnership bool) {
	rowAttrs := fmt.Sprintf(`data-mutant-row data-status="%s" data-survivor="%t" data-actionability="%s" data-operator="%s" data-risk="%s" data-group="%s" data-history="%s" data-age="%s" data-timing="%s" data-search="%s"`,
		html.EscapeString(row.Status),
		row.Survivor,
		html.EscapeString(row.Actionability),
		html.EscapeString(row.Operator),
		html.EscapeString(row.EquivalentRisk),
		html.EscapeString(row.GroupFilter),
		html.EscapeString(row.HistoryStatus),
		html.EscapeString(row.AgeBand),
		html.EscapeString(row.TimingBand),
		html.EscapeString(row.Search),
	)
	if hasOwnership {
		rowAttrs += fmt.Sprintf(` data-owner="%s" data-team="%s"`,
			html.EscapeString(row.Owner),
			html.EscapeString(row.Team),
		)
	}
	fmt.Fprintf(b, "<tr %s>", rowAttrs)
	b.WriteString("<td>")
	if row.SurvivorRank > 0 {
		fmt.Fprintf(b, `<span class="badge">#%d</span>`, row.SurvivorRank)
	} else {
		b.WriteString(`<span class="badge">-</span>`)
	}
	b.WriteString("</td><td><div class=\"mutant-main\">")
	fmt.Fprintf(b, `<div class="mutant-file">%s:%d</div>`, html.EscapeString(row.File), row.Line)
	if row.Function != "" {
		fmt.Fprintf(b, `<div class="mutant-meta">function=%s</div>`, html.EscapeString(row.Function))
	}
	fmt.Fprintf(b, `<div class="mutant-swap">%s -> %s</div>`, html.EscapeString(row.Original), html.EscapeString(row.Mutated))
	fmt.Fprintf(b, `<div class="mutant-id">%s</div>`, html.EscapeString(row.MutantID))
	if row.Description != "" {
		fmt.Fprintf(b, `<div class="mutant-meta">%s</div>`, html.EscapeString(row.Description))
	}
	b.WriteString("</div></td><td>")
	fmt.Fprintf(b, `<span class="badge badge-%s">%s</span>`, html.EscapeString(row.Status), html.EscapeString(strings.ReplaceAll(row.Status, "_", " ")))
	if row.FailureKind != "" {
		fmt.Fprintf(b, `<div class="mutant-meta">failure=%s</div>`, html.EscapeString(row.FailureKind))
	}
	b.WriteString("</td><td>")
	fmt.Fprintf(b, `<div class="mutant-meta">actionability=%s</div>`, html.EscapeString(row.Actionability))
	fmt.Fprintf(b, `<div class="mutant-meta">operator=%s</div>`, html.EscapeString(row.Operator))
	fmt.Fprintf(b, `<div class="mutant-meta">equivalent_risk=%s</div>`, html.EscapeString(row.EquivalentRisk))
	fmt.Fprintf(b, `<div class="mutant-meta">group=%s</div>`, html.EscapeString(row.GroupLabel))
	if row.PlatformSensitive {
		b.WriteString(`<div class="mutant-meta">platform-sensitive</div>`)
	}
	if row.NonProgressRisk != "" {
		fmt.Fprintf(b, `<div class="mutant-meta">non_progress_risk=%s</div>`, html.EscapeString(row.NonProgressRisk))
	}
	if row.OwnershipSummary != "" {
		fmt.Fprintf(b, `<div class="mutant-meta">ownership=%s</div>`, html.EscapeString(row.OwnershipSummary))
	}
	if row.SuggestedScope != "" {
		fmt.Fprintf(b, `<div class="mutant-meta">suggested_scope=%s</div>`, html.EscapeString(row.SuggestedScope))
	}
	if row.RecommendationPrimaryTest != "" {
		fmt.Fprintf(b, `<div class="mutant-meta">next_test=%s</div>`, html.EscapeString(row.RecommendationPrimaryTest))
	}
	if row.RecommendationStrategy != "" {
		fmt.Fprintf(b, `<div class="mutant-meta">test_strategy=%s</div>`, html.EscapeString(row.RecommendationStrategy))
	}
	if row.NearestTests != "" {
		fmt.Fprintf(b, `<div class="mutant-meta">nearby_tests=%s</div>`, html.EscapeString(row.NearestTests))
	}
	b.WriteString("</td><td>")
	fmt.Fprintf(b, `<div class="mutant-meta">history=%s</div>`, html.EscapeString(strings.ReplaceAll(row.HistoryStatus, "_", " ")))
	fmt.Fprintf(b, `<div class="mutant-meta">age=%s</div>`, html.EscapeString(row.AgeLabel))
	fmt.Fprintf(b, `<div class="mutant-meta">duration=%s</div>`, html.EscapeString(row.DurationText))
	fmt.Fprintf(b, `<div class="mutant-meta">timing_signal=%s</div>`, html.EscapeString(row.TimingLabel))
	b.WriteString("</td><td>")
	if row.StatusReason != "" {
		fmt.Fprintf(b, `<div class="mutant-reason">%s</div>`, html.EscapeString(row.StatusReason))
	}
	if row.RankReason != "" {
		fmt.Fprintf(b, `<div class="mutant-meta">rank=%s</div>`, html.EscapeString(row.RankReason))
	}
	if row.RecommendationSummary != "" {
		fmt.Fprintf(b, `<div class="mutant-meta">next_test_summary=%s</div>`, html.EscapeString(row.RecommendationSummary))
	}
	if row.RecommendationAssertions != "" {
		fmt.Fprintf(b, `<div class="mutant-meta">suggested_assertions=%s</div>`, html.EscapeString(row.RecommendationAssertions))
	}
	if row.SuggestedSkip != "" {
		fmt.Fprintf(b, `<div class="mutant-meta">skip=%s</div>`, html.EscapeString(row.SuggestedSkip))
	}
	b.WriteString("</td><td class=\"diff-shell\"><details><summary>Show diff</summary><pre>")
	b.WriteString(html.EscapeString(row.Diff))
	b.WriteString("</pre></details></td></tr>")
}

func renderHTMLSummary(b *strings.Builder, page htmlPageModel) {
	b.WriteString(`<section class="summary-shell"><details><summary>Raw summary</summary><pre>`)
	b.WriteString(html.EscapeString(page.RawSummary))
	b.WriteString(`</pre></details></section>`)
}

func renderHTMLScript(b *strings.Builder, page htmlPageModel) {
	b.WriteString(`<script>
(function(){
  const rows = Array.from(document.querySelectorAll('[data-mutant-row]'));
  const search = document.getElementById('filter-search');
  const status = document.getElementById('filter-status');
  const actionability = document.getElementById('filter-actionability');
  const operator = document.getElementById('filter-operator');
  const risk = document.getElementById('filter-risk');
  const group = document.getElementById('filter-group');
`)
	if page.HasOwnership {
		b.WriteString("  const owner = document.getElementById('filter-owner');\n")
		b.WriteString("  const team = document.getElementById('filter-team');\n")
	}
	b.WriteString(`  const history = document.getElementById('filter-history');
  const age = document.getElementById('filter-age');
  const timing = document.getElementById('filter-timing');
  const survivorsOnly = document.getElementById('filter-survivors-only');
  const reset = document.getElementById('filter-reset');
  const visibleCount = document.getElementById('visible-count');
  const totalCount = document.getElementById('total-count');
  const visibleSurvivors = document.getElementById('visible-survivors');
  const visibleGroups = document.getElementById('visible-groups');
  totalCount.textContent = String(rows.length);

  function matches(control, value) {
    return control.value === 'all' || control.value === value;
  }
`)
	if page.HasOwnership {
		b.WriteString(`
  function matchesOptional(control, value) {
    return !control || matches(control, value || '');
  }
`)
	}
	b.WriteString(`
  function normalize(value) {
    return (value || '').toLowerCase();
  }

  function applyFilters() {
    const term = normalize(search.value).trim();
    let visible = 0;
    let survivors = 0;
    const groups = new Set();

    rows.forEach((row) => {
      const data = row.dataset;
      const show =
        matches(status, data.status) &&
        matches(actionability, data.actionability) &&
        matches(operator, data.operator) &&
        matches(risk, data.risk) &&
        matches(group, data.group) &&
`)
	if page.HasOwnership {
		b.WriteString("        matchesOptional(owner, data.owner) &&\n")
		b.WriteString("        matchesOptional(team, data.team) &&\n")
	}
	b.WriteString(`        matches(history, data.history) &&
        matches(age, data.age) &&
        matches(timing, data.timing) &&
        (!survivorsOnly.checked || data.survivor === 'true') &&
        (term === '' || normalize(data.search).includes(term));

      row.hidden = !show;
      if (!show) {
        return;
      }
      visible += 1;
      if (data.survivor === 'true') {
        survivors += 1;
      }
      if (data.group && data.group !== 'ungrouped') {
        groups.add(data.group);
      }
    });

    visibleCount.textContent = String(visible);
    visibleSurvivors.textContent = String(survivors);
    visibleGroups.textContent = String(groups.size);
  }

`)
	if page.HasOwnership {
		b.WriteString("  [search, status, actionability, operator, risk, group, owner, team, history, age, timing].filter(Boolean).forEach((control) => {\n")
	} else {
		b.WriteString("  [search, status, actionability, operator, risk, group, history, age, timing].forEach((control) => {\n")
	}
	b.WriteString(`    control.addEventListener('input', applyFilters);
    control.addEventListener('change', applyFilters);
  });
  survivorsOnly.addEventListener('change', applyFilters);

  reset.addEventListener('click', function() {
    search.value = '';
`)
	if page.HasOwnership {
		b.WriteString("    [status, actionability, operator, risk, group, owner, team, history, age, timing].filter(Boolean).forEach((control) => {\n")
	} else {
		b.WriteString("    [status, actionability, operator, risk, group, history, age, timing].forEach((control) => {\n")
	}
	b.WriteString(`      control.value = 'all';
    });
    survivorsOnly.checked = true;
    applyFilters();
  });

  Array.from(document.querySelectorAll('[data-filter-target]')).forEach((button) => {
    button.addEventListener('click', function() {
      const target = document.getElementById(button.dataset.filterTarget);
      if (!target) {
        return;
      }
      target.value = button.dataset.filterValue || 'all';
      if (target.id === 'filter-group' || target.id === 'filter-operator') {
        survivorsOnly.checked = true;
      }
      applyFilters();
    });
  });

  applyFilters();
})();
</script>`)
}

func buildHTMLRows(result engine.RunResult) []htmlReportRow {
	sorted := append([]engine.MutantResult{}, result.Mutants...)
	sort.SliceStable(sorted, func(i, j int) bool {
		leftSurvivor := sorted[i].Status == engine.StatusSurvived
		rightSurvivor := sorted[j].Status == engine.StatusSurvived
		if leftSurvivor != rightSurvivor {
			return leftSurvivor
		}
		if leftSurvivor && rightSurvivor {
			leftRank := sorted[i].SurvivorRank
			rightRank := sorted[j].SurvivorRank
			if leftRank == 0 {
				leftRank = 1 << 20
			}
			if rightRank == 0 {
				rightRank = 1 << 20
			}
			if leftRank != rightRank {
				return leftRank < rightRank
			}
		}
		if sorted[i].Mutant.File != sorted[j].Mutant.File {
			return sorted[i].Mutant.File < sorted[j].Mutant.File
		}
		if sorted[i].Mutant.Line != sorted[j].Mutant.Line {
			return sorted[i].Mutant.Line < sorted[j].Mutant.Line
		}
		return sorted[i].MutantID < sorted[j].MutantID
	})

	rows := make([]htmlReportRow, 0, len(sorted))
	for _, mutant := range sorted {
		ageBand, ageLabel := htmlAgeBand(mutant.SurvivorAgeRuns)
		timingBand, timingLabel := htmlTimingBand(mutant.Duration)
		groupFilter := mutant.Mutant.GroupLabel
		groupLabel := mutant.Mutant.GroupLabel
		if strings.TrimSpace(groupFilter) == "" {
			groupFilter = "ungrouped"
			groupLabel = "ungrouped"
		}
		actionability := mutant.Actionability
		if strings.TrimSpace(actionability) == "" {
			actionability = "unknown"
		}
		equivalentRisk := mutant.Mutant.EquivalentRisk
		if strings.TrimSpace(equivalentRisk) == "" {
			equivalentRisk = "unknown"
		}
		historyStatus := mutant.HistoryStatus
		if strings.TrimSpace(historyStatus) == "" {
			historyStatus = "unknown"
		}
		rows = append(rows, htmlReportRow{
			MutantID:                  mutant.MutantID,
			Status:                    string(mutant.Status),
			Survivor:                  mutant.Status == engine.StatusSurvived,
			SurvivorRank:              mutant.SurvivorRank,
			Actionability:             actionability,
			Operator:                  mutant.Mutant.Operator,
			EquivalentRisk:            equivalentRisk,
			GroupFilter:               groupFilter,
			GroupLabel:                groupLabel,
			HistoryStatus:             historyStatus,
			AgeBand:                   ageBand,
			AgeLabel:                  ageLabel,
			TimingBand:                timingBand,
			TimingLabel:               timingLabel,
			DurationText:              htmlDurationText(mutant.Duration),
			File:                      mutant.Mutant.File,
			Line:                      mutant.Mutant.Line,
			Function:                  mutant.Mutant.Function,
			Original:                  mutant.Mutant.Original,
			Mutated:                   mutant.Mutant.Mutated,
			Description:               mutant.Mutant.Description,
			FailureKind:               mutant.FailureKind,
			StatusReason:              mutant.StatusReason,
			SuggestedSkip:             ledgerSuggestedReason(mutant, ""),
			SuggestedScope:            mutant.SuggestedTestScope,
			NearestTests:              strings.Join(mutant.NearestTests, ", "),
			RecommendationSummary:     recommendationSummary(mutant.TestRecommendation),
			RecommendationStrategy:    recommendationStrategy(mutant.TestRecommendation),
			RecommendationPrimaryTest: recommendationPrimaryTest(mutant.TestRecommendation),
			RecommendationAssertions:  recommendationAssertions(mutant.TestRecommendation),
			RankReason:                mutant.RankReason,
			Diff:                      mutant.Mutant.Diff,
			Actionable:                isActionableSurvivor(result.Environment.OS, mutant),
			PlatformSensitive:         mutant.Mutant.PlatformSensitive,
			NonProgressRisk:           mutant.Mutant.NonProgressRisk,
			Owner:                     ownershipRouteOwner(mutant.Mutant.Ownership),
			Team:                      ownershipRouteTeam(mutant.Mutant.Ownership),
			OwnershipSummary:          ownershipRouteSummary(mutant.Mutant.Ownership),
			Search:                    htmlSearchText(mutant, groupLabel, historyStatus, ageLabel, timingLabel),
		})
	}
	return rows
}

func buildHTMLSummaryModel(rows []htmlReportRow) htmlSummaryModel {
	actionableSurvivors, survivorGroups, longStandingSurvivors, slowSignals := htmlSummaryMetrics(rows)
	return htmlSummaryModel{
		ActionableSurvivors: actionableSurvivors,
		SurvivorGroups:      survivorGroups,
		LongStanding:        longStandingSurvivors,
		SlowSignals:         slowSignals,
	}
}

func initialHTMLVisibility(rows []htmlReportRow) (visible int, groupCount int) {
	groups := map[string]bool{}
	for _, row := range rows {
		if !row.Survivor {
			continue
		}
		visible++
		if row.GroupFilter != "ungrouped" {
			groups[row.GroupFilter] = true
		}
	}
	return visible, len(groups)
}

func htmlFilterOptions(rows []htmlReportRow, picker func(htmlReportRow) (string, string)) []htmlFilterOption {
	counts := map[string]int{}
	labels := map[string]string{}
	for _, row := range rows {
		value, label := picker(row)
		value = strings.TrimSpace(value)
		label = strings.TrimSpace(label)
		if value == "" || label == "" {
			continue
		}
		counts[value]++
		labels[value] = label
	}
	options := make([]htmlFilterOption, 0, len(counts))
	for value, count := range counts {
		options = append(options, htmlFilterOption{Value: value, Label: labels[value], Count: count})
	}
	sort.Slice(options, func(i, j int) bool {
		return options[i].Label < options[j].Label
	})
	return options
}

func htmlSummaryMetrics(rows []htmlReportRow) (actionableSurvivors, survivorGroups, longStandingSurvivors, slowSignals int) {
	groups := map[string]bool{}
	for _, row := range rows {
		if row.Survivor && row.Actionable {
			actionableSurvivors++
		}
		if row.Survivor && row.GroupFilter != "ungrouped" {
			groups[row.GroupFilter] = true
		}
		if row.Survivor && row.AgeBand == "long-standing" {
			longStandingSurvivors++
		}
		if row.TimingBand == "slow" {
			slowSignals++
		}
	}
	return actionableSurvivors, len(groups), longStandingSurvivors, slowSignals
}

func htmlAgeBand(runs int) (string, string) {
	switch {
	case runs <= 0:
		return "unknown", "unknown"
	case runs == 1:
		return "new", "new (1 run)"
	case runs < 5:
		return "aging", "aging (2-4 runs)"
	default:
		return "long-standing", "long-standing (5+ runs)"
	}
}

func htmlTimingBand(duration time.Duration) (string, string) {
	switch {
	case duration <= 0:
		return "unknown", "not recorded"
	case duration < 500*time.Millisecond:
		return "fast", "fast (<500ms)"
	case duration < 2*time.Second:
		return "medium", "medium (0.5-2s)"
	default:
		return "slow", "slow (>2s)"
	}
}

func htmlDurationText(duration time.Duration) string {
	if duration <= 0 {
		return "not recorded"
	}
	return duration.Round(time.Millisecond).String()
}

func htmlSearchText(mutant engine.MutantResult, groupLabel, historyStatus, ageLabel, timingLabel string) string {
	parts := []string{
		mutant.MutantID,
		mutant.Mutant.File,
		mutant.Mutant.Function,
		mutant.Mutant.Operator,
		mutant.Mutant.Description,
		mutant.StatusReason,
		mutant.RankReason,
		recommendationSummary(mutant.TestRecommendation),
		recommendationStrategy(mutant.TestRecommendation),
		recommendationPrimaryTest(mutant.TestRecommendation),
		recommendationAssertions(mutant.TestRecommendation),
		groupLabel,
		historyStatus,
		ageLabel,
		timingLabel,
		mutant.SuggestedTestScope,
		mutant.SuggestedSkipReason,
		strings.Join(mutant.NearestTests, " "),
	}
	if ownership := ownershipRouteSearch(mutant.Mutant.Ownership); ownership != "" {
		parts = append(parts, ownership)
	}
	return strings.Join(parts, " ")
}

func writeHTMLCard(b *strings.Builder, label string, value any) {
	fmt.Fprintf(b, `<div class="card"><span class="card-label">%s</span><span class="card-value">%v</span></div>`, html.EscapeString(label), value)
}

func writeHTMLInput(b *strings.Builder, id, label, placeholder string) {
	fmt.Fprintf(b, `<label for="%s"><span>%s</span><input id="%s" type="search" placeholder="%s"></label>`,
		html.EscapeString(id),
		html.EscapeString(label),
		html.EscapeString(id),
		html.EscapeString(placeholder),
	)
}

func writeHTMLSelect(b *strings.Builder, id, label string, options []htmlFilterOption, allLabel string) {
	fmt.Fprintf(b, `<label for="%s"><span>%s</span><select id="%s"><option value="all">%s</option>`,
		html.EscapeString(id),
		html.EscapeString(label),
		html.EscapeString(id),
		html.EscapeString(allLabel),
	)
	for _, option := range options {
		fmt.Fprintf(b, `<option value="%s">%s (%d)</option>`,
			html.EscapeString(option.Value),
			html.EscapeString(option.Label),
			option.Count,
		)
	}
	b.WriteString(`</select></label>`)
}

func writeHTMLQuickFilters(b *strings.Builder, label, target string, options []htmlFilterOption) {
	if len(options) == 0 {
		return
	}
	fmt.Fprintf(b, `<div class="quick-nav"><div class="chip-row"><strong>%s</strong>`, html.EscapeString(label))
	for _, option := range options {
		fmt.Fprintf(b, `<button class="quick-filter" type="button" data-filter-target="%s" data-filter-value="%s">%s (%d)</button>`,
			html.EscapeString(target),
			html.EscapeString(option.Value),
			html.EscapeString(option.Label),
			option.Count,
		)
	}
	b.WriteString(`</div></div>`)
}

func topHTMLFilterOptions(options []htmlFilterOption, limit int) []htmlFilterOption {
	if len(options) == 0 || limit <= 0 {
		return nil
	}
	ordered := append([]htmlFilterOption{}, options...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].Count != ordered[j].Count {
			return ordered[i].Count > ordered[j].Count
		}
		return ordered[i].Label < ordered[j].Label
	})
	if len(ordered) > limit {
		ordered = ordered[:limit]
	}
	return ordered
}
