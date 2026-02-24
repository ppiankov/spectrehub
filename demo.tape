# SpectreHub demo — record with: vhs demo.tape
# Produces demo.gif for spectrehub.dev hero section
# Install vhs: brew install charmbracelet/tap/vhs

Output demo.gif

Set Shell "bash"
Set FontSize 14
Set Width 1000
Set Height 600
Set Theme "Dracula"
Set TypingSpeed 40ms
Set Padding 20

# --- Doctor: verify setup ---
Type "spectrehub doctor"
Enter
Sleep 500ms

# Simulated doctor output (real command requires live API)
Type@0ms "  ✓ config              ~/.spectrehub.yaml"
Enter
Type@0ms "  ✓ license             team (up to 5 repos)"
Enter
Type@0ms "  ✓ api                 https://api.spectrehub.dev"
Enter
Type@0ms "  ✓ repo                acme-corp/infra"
Enter
Type@0ms "  ✓ vaultspectre        ready"
Enter
Type@0ms "  ✓ pgspectre           ready"
Enter
Type@0ms "  ✓ kafkaspectre        ready"
Enter
Type@0ms "  ✓ clickspectre        ready"
Enter
Type@0ms "  ✓ storage             .spectre"
Enter
Type@0ms ""
Enter
Type@0ms "all checks passed"
Enter
Sleep 2s

# --- Run: full audit cycle ---
Type "spectrehub run --store --repo acme-corp/infra"
Enter
Sleep 500ms

Type@0ms "Discovered 4 tool(s), 4 runnable"
Enter
Type@0ms ""
Enter
Type@0ms "  vaultspectre   ✓ ready"
Enter
Type@0ms "  pgspectre      ✓ ready"
Enter
Type@0ms "  kafkaspectre   ✓ ready"
Enter
Type@0ms "  clickspectre   ✓ ready"
Enter
Sleep 1s

Type@0ms ""
Enter
Type@0ms "Total Issues: 40"
Enter
Type@0ms "Health Score: GOOD (91%)"
Enter
Type@0ms ""
Enter
Type@0ms "Recommendations:"
Enter
Type@0ms "  1. [CRITICAL] Fix 8 missing Vault secrets"
Enter
Type@0ms "  2. [HIGH] Clean up 7 unused Kafka topics"
Enter
Type@0ms "  3. [MEDIUM] Review 18 stale ClickHouse tables"
Enter
Type@0ms ""
Enter
Type@0ms "Report synced to SpectreHub API (acme-corp/infra)"
Enter
Sleep 2s

# --- Explain score: show the math ---
Type "spectrehub explain-score"
Enter
Sleep 500ms

Type@0ms "Health Score Breakdown"
Enter
Type@0ms "======================"
Enter
Type@0ms ""
Enter
Type@0ms "1. Resources per tool:"
Enter
Type@0ms "   clickspectre   24 resources, 18 issues, 12 affected"
Enter
Type@0ms "   kafkaspectre   35 resources, 7 issues, 7 affected"
Enter
Type@0ms "   pgspectre      42 resources, 7 issues, 5 affected"
Enter
Type@0ms "   vaultspectre   40 resources, 8 issues, 8 affected"
Enter
Type@0ms "                  141 resources total"
Enter
Type@0ms ""
Enter
Type@0ms "2. Affected resources: 32 distinct"
Enter
Type@0ms ""
Enter
Type@0ms "3. Formula:"
Enter
Type@0ms "   score = (total - affected) / total * 100"
Enter
Type@0ms "   score = (141 - 13) / 141 * 100 = 90.8"
Enter
Type@0ms ""
Enter
Type@0ms "4. Thresholds:"
Enter
Type@0ms "     ≥ 95%  excellent"
Enter
Type@0ms "   → ≥ 85%  good"
Enter
Type@0ms "     ≥ 70%  warning"
Enter
Type@0ms "     ≥ 50%  critical"
Enter
Type@0ms "     ≥ 0%   severe"
Enter
Type@0ms ""
Enter
Type@0ms "Result: GOOD (90.8%)"
Enter
Sleep 3s
