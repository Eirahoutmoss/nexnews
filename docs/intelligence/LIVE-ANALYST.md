# NexNews Live Analyst

## Run

Start the Intelligence UI from the repository root:

```powershell
go run ./cmd/nexnews-intelligence-ui -db "$env:APPDATA\NexNews\nexnews-intelligence.json" -addr 127.0.0.1:8787
```

Open `http://127.0.0.1:8787/analyst`. One research question is the only required input.

## What happens on a run

1. Queries Google News RSS in Turkish and English plus the public GDELT DOC article search in parallel.
2. Merges exact duplicate URLs after removing common tracking parameters; preserves which search providers returned each record.
3. Retains publisher domain, available source date, title, URL and accessible short summary. It does not claim to have read the full article when only a headline/summary was retrieved.
4. Produces source coverage statistics, recurring term signals, possible cross-domain headline-copy signals, provider failures and follow-up questions.
5. If an optional OpenAI-compatible chat model is configured, builds an additional Turkish synthesis: main findings, conflicting signals, alternative explanations, timeline, open questions and next steps. Every returned source ID is checked against the collected article set. Model findings without a valid source ID are discarded.

The heuristic pass is available without an AI API key. Without a configured model, the report explicitly labels itself as rule-based and does not pretend to be deep semantic analysis.

## Optional model synthesis

Environment variables are used so secrets are not stored in source files:

- `NEXNEWS_ANALYST_API_KEY`: bearer token for an OpenAI-compatible chat-completions endpoint.
- `NEXNEWS_ANALYST_API_URL`: optional endpoint URL. If unset and an API key exists, defaults to `https://api.openai.com/v1/chat/completions`. Setting a local endpoint explicitly allows a local service such as an OpenAI-compatible Ollama endpoint without an API key.
- `NEXNEWS_ANALYST_MODEL`: optional model identifier; default is `gpt-4.1-mini`.

PowerShell example:

```powershell
$env:NEXNEWS_ANALYST_API_KEY = "your-key"
$env:NEXNEWS_ANALYST_MODEL = "your-model"
go run ./cmd/nexnews-intelligence-ui -db "$env:APPDATA\NexNews\nexnews-intelligence.json" -addr 127.0.0.1:8787
```

Use a model name supported by the endpoint you configured. The model request includes the research query plus source titles, domains, URLs, publication dates (when available) and short summaries, not full article downloads. This can incur external model-provider charges and sends that source metadata to the configured endpoint.

## Interpretation and limits

- Search-index coverage changes over time; source-provider failures are displayed alongside results.
- Distinct domains are not proof of independent reporting, and similar headlines are a review signal rather than evidence of copying.
- URL deduplication cannot reliably merge different Google News redirect URLs that point to the same underlying publisher story.
- Model citations are restricted to collected source IDs, but that does not by itself prove the model interpreted a cited source correctly or that a claim is true.
- A news-search engine is not a substitute for source-document verification. Legal, security, geopolitical or causal claims require human review of the original source.
- The first version does not automatically infer intent, causal relationships, future outcomes or statistical significance.

## Tests

```powershell
go test ./intelligence ./cmd/nexnews-intelligence-ui
```
