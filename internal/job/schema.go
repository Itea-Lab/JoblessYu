package job

// jobMetaSchemaDescription is the JSON shape the Groq extractor must
// return. Embedded in the system prompt because qwen/qwen3.6-27b does
// NOT support Groq's strict JSON schema mode (response_format with
// json_schema + strict: true). Instead we use json_object mode + describe
// the schema in the prompt + validate/retry on the client side.
const jobMetaSchemaDescription = `{
  "level":   "string — one of: Intern, Fresher, Junior, Senior, Unknown",
  "type":    "string — one of: Fulltime, Parttime, Contract, Internship, Unknown",
  "tags":    "object — category name → array of canonical tag strings. Categories: Cloud, IaC, Pipeline, Containers, Security, Languages, Data/DB, AI. Empty object {} if no tags found.",
  "salary":  "string — raw salary range if mentioned, else empty string",
  "remote":  "boolean — true only if JD explicitly says remote/work from home/hybrid",
  "summary": "string — max 200 chars, neutral tone, one-sentence JD summary"
}`
