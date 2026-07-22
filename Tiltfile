# Load the restart_process extension
load('ext://restart_process', 'docker_build_with_restart')

# Tilt owns the full local Kubernetes stack (namespace dev):
#   deploy/kubernetes/development/  — infra + all services
#   deploy/tilt/migrate-up.sh         — DB migrations via localhost:5432 port-forward
#   deploy/tilt/livekit-up.sh         — host Docker LiveKit + SIP + trunk/dispatch
#
# Primary OSS path (no cluster): Docker Compose / Codespaces — see docs/local-setup.md
#
# Local k8s day-to-day:
#   ./deploy/tilt/preflight.sh && tilt up
#
# Optional cluster packaging: deploy/helm/ + deploy/helm/values/dev.yaml
#
# Tilt docker_build accepts one platform string (not comma-separated multi-arch).
# Default amd64; on Apple Silicon local clusters you may need TILT_DOCKER_PLATFORM=linux/arm64.
_docker_platform = os.getenv('TILT_DOCKER_PLATFORM', 'linux/amd64')
# Bare image names (api-gateway, web, …) — local Docker / cluster, no remote registry.

# Deploy into the dev namespace (same as Helm / Postgres / Redis)
k8s_namespace('dev')

# Allow the currently selected kube context (Docker Desktop, Minikube, kind, etc.)
allow_k8s_contexts(k8s_context())

### K8s Config ###

# Kustomize sets namespace: dev on every manifest (k8s_namespace alone is not always enough).
k8s_yaml(kustomize('./deploy/kubernetes/development'))

# Infra must be up before app pods connect to postgres/redis/rabbitmq.
infra_deps = ['postgres', 'redis', 'rabbitmq']
# App services wait for SQL migrations (local_resource below).
app_deps = infra_deps + ['db-migrate']

local_resource(
  'tilt-preflight',
  './deploy/tilt/preflight.sh',
  labels='setup',
  auto_init=True,
)

k8s_resource(
  'postgres',
  port_forwards='5432:5432',
  labels='infra',
)
k8s_resource('redis', port_forwards='6379:6379', labels='infra')
k8s_resource(
  'rabbitmq',
  port_forwards=['5672:5672', '15672:15672'],
  labels='infra',
)
k8s_resource('jaeger', port_forwards='16686:16686', labels='infra')

# LiveKit + SIP on host Docker (not in-cluster). Starts compose and provisions
# inbound trunk + agent dispatch rule (see deploy/tilt/livekit-up.sh).
local_resource(
  'livekit-docker',
  './deploy/tilt/livekit-up.sh',
  deps=[
    './deploy/docker/livekit',
    './deploy/tilt/livekit-up.sh',
  ],
  labels='infra',
  auto_init=True,
)

local_resource(
  'db-migrate',
  './deploy/tilt/migrate-up.sh',
  deps=['./migrations', './Makefile', './deploy/tilt/migrate-up.sh'],
  resource_deps=['postgres'],
  labels='database',
  auto_init=True,
)

### API Gateway ###

gateway_compile_cmd = 'CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/api-gateway ./services/api-gateway/cmd/api-gateway'
if os.name == 'nt':
  gateway_compile_cmd = './deploy/docker/development/api-gateway-build.bat'

local_resource(
  'api-gateway-compile',
  gateway_compile_cmd,
  deps=['./services/api-gateway', './shared'], labels="compiles")

docker_build_with_restart(
  'api-gateway',
  '.',
  entrypoint=['/app/build/api-gateway'],
  dockerfile='./deploy/docker/development/api-gateway.Dockerfile',
  platform=_docker_platform,
  only=[
    './build/api-gateway',
    './shared',
    './services/api-gateway',
    './go.mod',
    './go.sum',
  ],
  live_update=[
    sync('./build', '/app/build'),
    sync('./shared', '/app/shared'),
  ],
)

# Bind 0.0.0.0 so Cursor Remote SSH / laptop browsers can reach the gateway
# (not only Tilt's default 127.0.0.1). Also forward 8081 in the Ports panel.
k8s_resource(
  'api-gateway',
  port_forwards=[port_forward(8081, 8081, name='api-gateway', host='0.0.0.0')],
  resource_deps=app_deps + ['api-gateway-compile'],
  labels="services",
)
### End of API Gateway ###

### Patient Service (HTTP Gateway) ###
docker_build(
  'patient-service',
  '.',
  dockerfile='./deploy/docker/development/patient-service.Dockerfile',
  platform=_docker_platform,
)

k8s_resource('patient-service', port_forwards=8083, resource_deps=app_deps + ['livekit-docker'], labels="services")
k8s_resource(
  'welfare-check-dispatcher',
  resource_deps=app_deps + ['patient-service', 'auth-service', 'audit-service'],
  labels='jobs',
)
### End of Patient Service ###


### Auth Service ###
docker_build(
  'auth-service',
  '.',
  dockerfile='./deploy/docker/development/auth-service.Dockerfile',
  platform=_docker_platform,
)

k8s_resource('auth-service', port_forwards='8082:9092', resource_deps=app_deps, labels="services")
### End of Auth Service ###

### Notification Service ###
docker_build(
  'notification-service',
  '.',
  dockerfile='./deploy/docker/development/notification-service.Dockerfile',
  platform=_docker_platform,
)

# Expose SMS webhook on all interfaces so VoIP.ms can reach the GCP VM public IP
# (open GCP firewall TCP 3001). Path: http://<VM_EXTERNAL_IP>:3001/sms?api_key=...
k8s_resource(
  'notification-service',
  port_forwards=[
    50056,
    port_forward(3001, 3001, name='sms-webhook', host='0.0.0.0'),
  ],
  resource_deps=app_deps,
  labels="services",
)
### End of Notification Service ###

### Health Records Service ###
docker_build(
  'health-records-service',
  '.',
  dockerfile='./deploy/docker/development/health-records-service.Dockerfile',
  platform=_docker_platform,
)

k8s_resource('health-records-service', port_forwards='50054:50054', resource_deps=app_deps, labels="services")
### End of Health Records Service ###

### Analytics Service ###
docker_build(
  'analytics-service',
  '.',
  dockerfile='./deploy/docker/development/analytics-service.Dockerfile',
  platform=_docker_platform,
)

k8s_resource(
  'analytics-service',
  port_forwards='50055:50054',
  resource_deps=app_deps,
  labels="services",
)
### End of Analytics Service ###

### Audit Service ###
docker_build(
  'audit-service',
  '.',
  dockerfile='./deploy/docker/development/audit-service.Dockerfile',
  platform=_docker_platform,
)

k8s_resource(
  'audit-service',
  port_forwards='50058:50058',
  resource_deps=app_deps,
  labels="services",
)
### End of Audit Service ###

### Location Service ###
docker_build(
  'location-service',
  '.',
  dockerfile='./deploy/docker/development/location-service.Dockerfile',
  platform=_docker_platform,
)

k8s_resource(
  'location-service',
  port_forwards=[
    50051,
    port_forward(8091, 8090, name='location-ws', host='0.0.0.0'),
  ],
  resource_deps=app_deps,
  labels="services",
)
### End of Location Service ###

### Translation Service ###
docker_build(
  'translation-service',
  '.',
  dockerfile='./deploy/docker/development/translation-service.Dockerfile',
  platform=_docker_platform,
)

k8s_resource(
  'translation-service',
  port_forwards='50057:50057',
  resource_deps=app_deps,
  labels="services",
)
### End of Translation Service ###

### Interpretation Service ###
docker_build(
  'interpretation-service',
  '.',
  dockerfile='./deploy/docker/development/interpretation-service.Dockerfile',
  platform=_docker_platform,
)

k8s_resource(
  'interpretation-service',
  port_forwards='8095:8095',
  resource_deps=app_deps + ['translation-service'],
  labels="services",
)
### End of Interpretation Service ###

### MCP Server ###
docker_build(
  'mcp-server',
  '.',
  dockerfile='./deploy/docker/development/mcp-server.Dockerfile',
  platform=_docker_platform,
)

k8s_resource(
  'mcp-server',
  port_forwards='8092:8092',
  resource_deps=app_deps,
  labels="services",
)
### End of MCP Server ###

### Voice Agent Service ###
docker_build(
  'voice-agent-service',
  './services/voice-agent-service',
  dockerfile='./services/voice-agent-service/Dockerfile',
  platform=_docker_platform,
)

k8s_resource(
  'voice-agent-service',
  port_forwards='8090:8090',
  resource_deps=app_deps + ['mcp-server', 'livekit-docker'],
  labels="services",
)
### End of Voice Agent Service ###

docker_build(
  'web',
  '.',
  dockerfile='./deploy/docker/development/web.Dockerfile',
  platform=_docker_platform,
  build_args={
    'NEXT_PUBLIC_API_URL': 'http://localhost:8081',
    'NEXT_PUBLIC_LOCATION_WS_URL': 'ws://localhost:8091',
  },
)

# Bind 0.0.0.0 so Cursor port-forward / local browser can open the Next app.
# Forward 3000 (UI) + 8081 (API) in Cursor Ports; open http://localhost:3000
k8s_resource(
  'web',
  port_forwards=[port_forward(3000, 3000, name='web', host='0.0.0.0')],
  resource_deps=app_deps,
  labels="frontend",
)

### End of Web Frontend ###

### Mobile Frontend ###
docker_build(
  'mobile',
  '.',
  dockerfile='./deploy/docker/development/mobile.Dockerfile',
  platform=_docker_platform,
  build_args={
    'EXPO_PUBLIC_API_URL': 'http://localhost:8081',
    'EXPO_PUBLIC_LOCATION_WS_URL': 'ws://localhost:8091',
  },
  only=[
    './mobile',
  ],
)

k8s_resource('mobile', port_forwards='8084:8084', resource_deps=app_deps, labels="frontend")

### End of Mobile Frontend ###
