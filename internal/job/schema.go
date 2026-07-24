package job

// jobMetaSchemaDescription is the JSON shape the Groq extractor must
// return. Embedded in the system prompt to reinforce the expected output
// shape alongside json_object response_format mode.
const jobMetaSchemaDescription = `{
  "level":   "string — one of: Intern, Fresher, Junior, Senior, Unknown",
  "type":    "string — one of: Fulltime, Parttime, Contract, Internship, Unknown",
  "tags":    "object — category name → array of canonical tag strings. Categories: Cloud, IaC, Pipeline, Containers, Security, Languages, Data/DB, AI. Empty object {} if no tags found.",
  "salary":  "string — raw salary range if mentioned, else empty string",
  "remote":  "boolean — true only if JD explicitly says remote/work from home/hybrid",
  "summary": "string — max 200 chars, neutral tone, one-sentence JD summary"
}`
