# lab-result-tools

Parses messy lab-result and reference-range strings into typed values, compares a
result against a reference range, converts between lab concentration units, and
validates/batch-evaluates full result records — built for the [Axiom](https://axiomide.com)
marketplace.

**This package is STRING/NUMERIC normalization and RANGE COMPARISON against
reference ranges the CALLER supplies. It is NOT a clinical decision system.**
It ships NO built-in medical reference ranges, NO analyte molar-mass table, and NO
clinical thresholds of any kind, and none of its outputs are medical advice. Every
threshold, range, and molar mass it uses is an explicit caller-supplied input — if a
computation would require inventing or assuming a clinical value, a node returns a
structured error or an `indeterminate` verdict instead of guessing.

## Use it from your agent or app

Every node in this package is a **live, auto-scaling API endpoint** on the
[Axiom](https://axiomide.com) marketplace — call it from an AI agent or your
own code, with nothing to self-host.

**📦 See it on the marketplace:**
https://dev.axiomide.com/marketplace/christiangeorgelucas/lab-result-tools@0.1.0

**Hook it up to an AI agent (MCP).** Add Axiom's hosted MCP server to any MCP
client and every node becomes a typed tool your agent can call — search the
catalog, inspect a schema, and invoke it directly.

```bash
# Claude Code
claude mcp add --transport http axiom https://api.axiomide.com/mcp \
  --header "Authorization: Bearer $AXIOM_API_KEY"
```

Claude Desktop, Cursor, or any config-based client:

```json
{
  "mcpServers": {
    "axiom": {
      "type": "http",
      "url": "https://api.axiomide.com/mcp",
      "headers": { "Authorization": "Bearer YOUR_AXIOM_API_KEY" }
    }
  }
}
```

**Call it from the CLI.**

```bash
axiom invoke christiangeorgelucas/lab-result-tools/ParseResultValue --input '{"text": "5.2 mg/dL"}'
```

**Call it over HTTP.**

```bash
curl -X POST https://api.axiomide.com/invocations/v1/nodes/christiangeorgelucas/lab-result-tools/0.1.0/ParseResultValue \
  -H "Authorization: Bearer $AXIOM_API_KEY" \
  -H 'Content-Type: application/json' \
  -d '{"text": "5.2 mg/dL"}'
```

### Get started free

Install the CLI:

```bash
# macOS / Linux — Homebrew
brew install axiomide/tap/axiom

# macOS / Linux — install script
curl -fsSL https://raw.githubusercontent.com/AxiomIDE/axiom-releases/main/install.sh | sh
```

**Windows:** download the `windows/amd64` `.zip` from the
[releases page](https://github.com/AxiomIDE/axiom-releases/releases), unzip
it, and put `axiom.exe` on your `PATH`.

Then `axiom version` to verify, `axiom login` (GitHub or Google) to
authenticate, and create an API key under **Console → API Keys**. Docs and
sign-up at **[axiomide.com](https://axiomide.com)**.

## Nodes

- **ParseResultValue** — parse a messy lab-result string ("<0.5", ">1000",
  "5.2 mg/dL", "NEGATIVE", "1,234.5", "3.4-3.9") into a typed value: `numeric`,
  `censored_low`, `censored_high`, `range`, or `qualitative`. A censored value
  ("<0.5") is modeled explicitly as a threshold with a direction, never
  silently coerced to the number it names.
- **ParseReferenceRange** — parse a reference-range string ("3.5-5.0", "<200",
  ">=40", "70 - 110 mg/dL") into explicit low/high bounds with their own
  inclusive/exclusive flags. A reversed range (low > high) is a structured
  error, not silently accepted.
- **EvaluateResult** — compare a parsed value against a parsed reference range
  (and an optional tighter critical range), returning
  `low`/`normal`/`high`/`critical_low`/`critical_high`, or `indeterminate` when
  the comparison cannot be soundly decided (a qualitative value against a
  numeric range, a censored value whose possible interval straddles a
  boundary, or mismatched units). Never guesses a pass/fail.
- **NormalizeResultUnits** — convert a lab concentration value between mass
  units (g, mg, µg, ng, pg per L/dL/mL/µL) and/or molar units (mol, mmol,
  µmol, nmol, pmol per L/dL/mL/µL). A mass<->molar conversion (e.g. mg/dL to
  mmol/L) REQUIRES the analyte's molar mass in g/mol as an explicit input —
  this node never assumes or looks one up. Out of scope: any unit shape
  outside lab mass/molar concentration (temperature, length, dimensionless
  ratios) — compose with
  [christiangeorgelucas/unit-tools](https://axiomide.com)' `Convert` node for
  those instead.
- **ValidateResultRecord** — validate one full lab result record (test name,
  value, unit, reference range, collection timestamp) and return a
  field-level list of errors/warnings plus the record's already-parsed value
  and range.
- **BatchEvaluate** — validate and evaluate many lab result records in one
  call, each against its own reference range, returning a per-record flag
  plus aggregate counts (normal/low/high/critical/indeterminate/invalid).

## License

MIT — see [LICENSE](LICENSE). Third-party dependency license in
[THIRD_PARTY_LICENSES.txt](THIRD_PARTY_LICENSES.txt).
