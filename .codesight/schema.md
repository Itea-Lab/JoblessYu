# Schema

### jobs
- id: bigint(auto) (pk)
- job_id: text (fk)
- site: text (required)
- job_url: text (required)
- title: text
- company: text
- location: text
- job_type: text
- description: text
- fetched_at: timestamp(tz) (required)
