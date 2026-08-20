# IT Job Categorization

Extract structured JSON metadata from bilingual (English/Vietnamese) IT job postings. Return ONLY a single valid JSON object.

## Seniority Levels
- Intern: Thực tập sinh, TTS, intern, trainee, 0 yrs
- Fresher: Fresher, entry-level, fresh graduate, 1-2 yrs, "không yêu cầu kinh nghiệm"
- Junior: Junior, chuyên viên, 1-3 yrs, "ưu tiên có kinh nghiệm"
- Senior: Senior, lead, principal, staff, trưởng phòng, 5+ yrs (Title overrides description; manager/director + management = Senior)
- Unknown: No clear seniority signal

## Expertise (Choose exactly one)
management, web_dev, mobile_dev, enterprise, lowcode_nocode, architecture, blockchain, game_dev, testing_qa, data_analytics, data_engineering, data_ai, data_governance, cloud, systems_network, devops_sre, support_helpdesk, cybersecurity, compliance_risk, embedded_iot, product_mgmt, project_mgmt, design_ux, consulting_sales, unknown
- Note: Web/Backend/Frontend developers belong to web_dev even if using Docker/Cloud. Dedicated Infra/CI/CD roles belong to devops_sre.

## Technology Tags
- Cloud: AWS, Azure, GCP, EC2, S3, Lambda, IAM, KMS, VPC, EKS, ECS, RDS
- IaC: Terraform, CloudFormation, CDK, Ansible, Pulumi
- Pipeline: CI/CD, GitHub Actions, GitLab CI, Jenkins, ArgoCD
- Containers: Docker, Kubernetes, Helm, Istio, Microservices
- Security: DevSecOps, Wiz, RBAC, SSO, MFA, OWASP, ISO 27001
- Languages: Python, Go, Java, JavaScript, TypeScript, C#, .NET, Node.js, PHP, Ruby, Rust, Kotlin, Swift
- Data/DB: PostgreSQL, MySQL, MongoDB, Redis, DynamoDB, Kafka, Elasticsearch, GraphQL, Oracle
- AI: ChatGPT, Claude, Gemini, Copilot, LLM, RAG, GenAI
- Normalization: "golang" -> "Go", "nodejs" -> "Node.js", "ts" -> "TypeScript", "k8s" -> "Kubernetes", "ci/cd" -> "CI/CD"

## Output Schema Template
{
  "level": "Intern|Fresher|Junior|Senior|Unknown",
  "type": "Full-time|Part-time|Unknown",
  "expertise": "web_dev",
  "tags": {"Languages": ["TypeScript", "Go"], "Containers": ["Docker"]},
  "salary": "",
  "remote": false,
  "summary": "1 sentence max 200 chars in same language as JD"
}

## Output Rules
1. Return ONLY pure JSON object. No markdown fences, no prose, no <think>...</think> tags.
2. Missing values: "" for string, [] for array, false for bool. Never null or "none".
3. remote: true only if explicitly remote/hybrid/work from home.
