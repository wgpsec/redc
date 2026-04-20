export namespace ai {
	
	export class RegistrySkill {
	    id: string;
	    name: string;
	    description: string;
	    tags: string[];
	    url: string;
	    sha256?: string;
	    installed?: boolean;
	    hasUpdate?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new RegistrySkill(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.tags = source["tags"];
	        this.url = source["url"];
	        this.sha256 = source["sha256"];
	        this.installed = source["installed"];
	        this.hasUpdate = source["hasUpdate"];
	    }
	}
	export class Skill {
	    id: string;
	    name: string;
	    description: string;
	    tags: string[];
	    content?: string;
	
	    static createFrom(source: any = {}) {
	        return new Skill(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.tags = source["tags"];
	        this.content = source["content"];
	    }
	}
	export class SkillIndex {
	    id: string;
	    name: string;
	    description: string;
	    tags: string[];
	
	    static createFrom(source: any = {}) {
	        return new SkillIndex(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.tags = source["tags"];
	    }
	}

}

export namespace cost {
	
	export class ProviderCostSummary {
	    provider: string;
	    total_hourly_cost: number;
	    total_monthly_cost: number;
	    currency: string;
	    resource_count: number;
	
	    static createFrom(source: any = {}) {
	        return new ProviderCostSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.provider = source["provider"];
	        this.total_hourly_cost = source["total_hourly_cost"];
	        this.total_monthly_cost = source["total_monthly_cost"];
	        this.currency = source["currency"];
	        this.resource_count = source["resource_count"];
	    }
	}
	export class ResourceCostBreakdown {
	    resource_type: string;
	    resource_name: string;
	    provider: string;
	    count: number;
	    unit_hourly: number;
	    unit_monthly: number;
	    total_hourly: number;
	    total_monthly: number;
	    currency: string;
	    available: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ResourceCostBreakdown(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.resource_type = source["resource_type"];
	        this.resource_name = source["resource_name"];
	        this.provider = source["provider"];
	        this.count = source["count"];
	        this.unit_hourly = source["unit_hourly"];
	        this.unit_monthly = source["unit_monthly"];
	        this.total_hourly = source["total_hourly"];
	        this.total_monthly = source["total_monthly"];
	        this.currency = source["currency"];
	        this.available = source["available"];
	    }
	}
	export class CostEstimate {
	    total_hourly_cost: number;
	    total_monthly_cost: number;
	    currency: string;
	    breakdown: ResourceCostBreakdown[];
	    provider_breakdown?: Record<string, ProviderCostSummary>;
	    unavailable_count: number;
	    timestamp: time.Time;
	    disclaimer: string;
	    warnings?: string[];
	
	    static createFrom(source: any = {}) {
	        return new CostEstimate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total_hourly_cost = source["total_hourly_cost"];
	        this.total_monthly_cost = source["total_monthly_cost"];
	        this.currency = source["currency"];
	        this.breakdown = this.convertValues(source["breakdown"], ResourceCostBreakdown);
	        this.provider_breakdown = this.convertValues(source["provider_breakdown"], ProviderCostSummary, true);
	        this.unavailable_count = source["unavailable_count"];
	        this.timestamp = this.convertValues(source["timestamp"], time.Time);
	        this.disclaimer = source["disclaimer"];
	        this.warnings = source["warnings"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	

}

export namespace main {
	
	export class AIChatMessage {
	    role: string;
	    content: string;
	
	    static createFrom(source: any = {}) {
	        return new AIChatMessage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.role = source["role"];
	        this.content = source["content"];
	    }
	}
	export class AuditLogResult {
	    entries: mod.AuditLogEntry[];
	    total: number;
	
	    static createFrom(source: any = {}) {
	        return new AuditLogResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entries = this.convertValues(source["entries"], mod.AuditLogEntry);
	        this.total = source["total"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class BalanceInfo {
	    provider: string;
	    amount: string;
	    currency: string;
	    updatedAt: string;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new BalanceInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.provider = source["provider"];
	        this.amount = source["amount"];
	        this.currency = source["currency"];
	        this.updatedAt = source["updatedAt"];
	        this.error = source["error"];
	    }
	}
	export class BillInfo {
	    provider: string;
	    month: string;
	    totalAmount: string;
	    currency: string;
	    startDate: string;
	    endDate: string;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new BillInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.provider = source["provider"];
	        this.month = source["month"];
	        this.totalAmount = source["totalAmount"];
	        this.currency = source["currency"];
	        this.startDate = source["startDate"];
	        this.endDate = source["endDate"];
	        this.error = source["error"];
	    }
	}
	export class CaseInfo {
	    id: string;
	    name: string;
	    type: string;
	    state: string;
	    stateTime: string;
	    createTime: string;
	    operator: string;
	    isSpotInstance: boolean;
	    tags: string[];
	    plugins?: string;
	
	    static createFrom(source: any = {}) {
	        return new CaseInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.type = source["type"];
	        this.state = source["state"];
	        this.stateTime = source["stateTime"];
	        this.createTime = source["createTime"];
	        this.operator = source["operator"];
	        this.isSpotInstance = source["isSpotInstance"];
	        this.tags = source["tags"];
	        this.plugins = source["plugins"];
	    }
	}
	export class ComposeServiceSummary {
	    name: string;
	    rawName: string;
	    template: string;
	    provider: string;
	    profiles: string[];
	    dependsOn: string[];
	    replicas: number;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new ComposeServiceSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.rawName = source["rawName"];
	        this.template = source["template"];
	        this.provider = source["provider"];
	        this.profiles = source["profiles"];
	        this.dependsOn = source["dependsOn"];
	        this.replicas = source["replicas"];
	        this.status = source["status"];
	    }
	}
	export class ComposeSummary {
	    file: string;
	    services: ComposeServiceSummary[];
	    total: number;
	
	    static createFrom(source: any = {}) {
	        return new ComposeSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.file = source["file"];
	        this.services = this.convertValues(source["services"], ComposeServiceSummary);
	        this.total = source["total"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ConfigInfo {
	    redcPath: string;
	    projectPath: string;
	    logPath: string;
	    httpProxy: string;
	    httpsProxy: string;
	    socks5Proxy: string;
	    noProxy: string;
	    debugEnabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ConfigInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.redcPath = source["redcPath"];
	        this.projectPath = source["projectPath"];
	        this.logPath = source["logPath"];
	        this.httpProxy = source["httpProxy"];
	        this.httpsProxy = source["httpsProxy"];
	        this.socks5Proxy = source["socks5Proxy"];
	        this.noProxy = source["noProxy"];
	        this.debugEnabled = source["debugEnabled"];
	    }
	}
	export class EndpointCheck {
	    name: string;
	    url: string;
	    ok: boolean;
	    status: number;
	    error: string;
	    latencyMs: number;
	    checkedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new EndpointCheck(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.url = source["url"];
	        this.ok = source["ok"];
	        this.status = source["status"];
	        this.error = source["error"];
	        this.latencyMs = source["latencyMs"];
	        this.checkedAt = source["checkedAt"];
	    }
	}
	export class ExecCommandResult {
	    stdout: string;
	    stderr: string;
	    exitCode: number;
	    success: boolean;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new ExecCommandResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.stdout = source["stdout"];
	        this.stderr = source["stderr"];
	        this.exitCode = source["exitCode"];
	        this.success = source["success"];
	        this.error = source["error"];
	    }
	}
	export class F8xInstallRecord {
	    id: string;
	    caseID: string;
	    flags: string;
	    status: string;
	    output: string;
	    startedAt: string;
	    finishedAt?: string;
	
	    static createFrom(source: any = {}) {
	        return new F8xInstallRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.caseID = source["caseID"];
	        this.flags = source["flags"];
	        this.status = source["status"];
	        this.output = source["output"];
	        this.startedAt = source["startedAt"];
	        this.finishedAt = source["finishedAt"];
	    }
	}
	export class F8xStatus {
	    deployed: boolean;
	    version?: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new F8xStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.deployed = source["deployed"];
	        this.version = source["version"];
	        this.error = source["error"];
	    }
	}
	export class FileTransferResult {
	    success: boolean;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new FileTransferResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.error = source["error"];
	    }
	}
	export class MCPStatus {
	    running: boolean;
	    mode: string;
	    address: string;
	    protocolVersion: string;
	
	    static createFrom(source: any = {}) {
	        return new MCPStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.running = source["running"];
	        this.mode = source["mode"];
	        this.address = source["address"];
	        this.protocolVersion = source["protocolVersion"];
	    }
	}
	export class OnboardingStatus {
	    credentialsConfigured: boolean;
	    templatesInstalled: boolean;
	    scenesCreated: boolean;
	    dismissed: boolean;
	
	    static createFrom(source: any = {}) {
	        return new OnboardingStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.credentialsConfigured = source["credentialsConfigured"];
	        this.templatesInstalled = source["templatesInstalled"];
	        this.scenesCreated = source["scenesCreated"];
	        this.dismissed = source["dismissed"];
	    }
	}
	export class OrchestratorConfig {
	    maxRounds: number;
	    objective: string;
	    autoApprove: boolean;
	
	    static createFrom(source: any = {}) {
	        return new OrchestratorConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.maxRounds = source["maxRounds"];
	        this.objective = source["objective"];
	        this.autoApprove = source["autoApprove"];
	    }
	}
	export class PlanEdge {
	    from: string;
	    to: string;
	
	    static createFrom(source: any = {}) {
	        return new PlanEdge(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.from = source["from"];
	        this.to = source["to"];
	    }
	}
	export class PlanTypeSummary {
	    type: string;
	    label: string;
	    count: number;
	    actions: string;
	
	    static createFrom(source: any = {}) {
	        return new PlanTypeSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.label = source["label"];
	        this.count = source["count"];
	        this.actions = source["actions"];
	    }
	}
	export class PlanResourceChange {
	    address: string;
	    type: string;
	    name: string;
	    providerName: string;
	    actions: string[];
	    isData: boolean;
	    detail?: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new PlanResourceChange(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.address = source["address"];
	        this.type = source["type"];
	        this.name = source["name"];
	        this.providerName = source["providerName"];
	        this.actions = source["actions"];
	        this.isData = source["isData"];
	        this.detail = source["detail"];
	    }
	}
	export class PlanPreview {
	    hasChanges: boolean;
	    toCreate: number;
	    toUpdate: number;
	    toDelete: number;
	    toRecreate: number;
	    isSpotInstance: boolean;
	    resources: PlanResourceChange[];
	    edges: PlanEdge[];
	    typeSummary: PlanTypeSummary[];
	
	    static createFrom(source: any = {}) {
	        return new PlanPreview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hasChanges = source["hasChanges"];
	        this.toCreate = source["toCreate"];
	        this.toUpdate = source["toUpdate"];
	        this.toDelete = source["toDelete"];
	        this.toRecreate = source["toRecreate"];
	        this.isSpotInstance = source["isSpotInstance"];
	        this.resources = this.convertValues(source["resources"], PlanResourceChange);
	        this.edges = this.convertValues(source["edges"], PlanEdge);
	        this.typeSummary = this.convertValues(source["typeSummary"], PlanTypeSummary);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	export class PluginInfo {
	    name: string;
	    version: string;
	    description: string;
	    description_en: string;
	    author: string;
	    homepage: string;
	    category: string;
	    tags: string[];
	    enabled: boolean;
	    dir: string;
	    config_schema?: Record<string, plugin.ConfigField>;
	    config?: Record<string, any>;
	
	    static createFrom(source: any = {}) {
	        return new PluginInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.version = source["version"];
	        this.description = source["description"];
	        this.description_en = source["description_en"];
	        this.author = source["author"];
	        this.homepage = source["homepage"];
	        this.category = source["category"];
	        this.tags = source["tags"];
	        this.enabled = source["enabled"];
	        this.dir = source["dir"];
	        this.config_schema = this.convertValues(source["config_schema"], plugin.ConfigField, true);
	        this.config = source["config"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PluginUpdateInfo {
	    name: string;
	    version: string;
	
	    static createFrom(source: any = {}) {
	        return new PluginUpdateInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.version = source["version"];
	    }
	}
	export class PortForwardInfo {
	    id: string;
	    caseId: string;
	    localPort: number;
	    remoteHost: string;
	    remotePort: number;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new PortForwardInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.caseId = source["caseId"];
	        this.localPort = source["localPort"];
	        this.remoteHost = source["remoteHost"];
	        this.remotePort = source["remotePort"];
	        this.status = source["status"];
	    }
	}
	export class ProjectInfo {
	    name: string;
	    path: string;
	    createTime: string;
	    user: string;
	
	    static createFrom(source: any = {}) {
	        return new ProjectInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.createTime = source["createTime"];
	        this.user = source["user"];
	    }
	}
	export class ProviderCredential {
	    name: string;
	    fields: Record<string, string>;
	    hasSecrets: Record<string, boolean>;
	
	    static createFrom(source: any = {}) {
	        return new ProviderCredential(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.fields = source["fields"];
	        this.hasSecrets = source["hasSecrets"];
	    }
	}
	export class ProvidersConfigInfo {
	    configPath: string;
	    providers: ProviderCredential[];
	
	    static createFrom(source: any = {}) {
	        return new ProvidersConfigInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.configPath = source["configPath"];
	        this.providers = this.convertValues(source["providers"], ProviderCredential);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class RegistryTemplate {
	    name: string;
	    description: string;
	    description_en: string;
	    author: string;
	    latest: string;
	    versions: string[];
	    updatedAt: string;
	    tags: string[];
	    installed: boolean;
	    localVersion: string;
	
	    static createFrom(source: any = {}) {
	        return new RegistryTemplate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.description = source["description"];
	        this.description_en = source["description_en"];
	        this.author = source["author"];
	        this.latest = source["latest"];
	        this.versions = source["versions"];
	        this.updatedAt = source["updatedAt"];
	        this.tags = source["tags"];
	        this.installed = source["installed"];
	        this.localVersion = source["localVersion"];
	    }
	}
	export class ResourceSummary {
	    type: string;
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new ResourceSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.count = source["count"];
	    }
	}
	export class TemplateInfo {
	    name: string;
	    description: string;
	    description_en: string;
	    version: string;
	    user: string;
	    module: string;
	    plugins: string;
	
	    static createFrom(source: any = {}) {
	        return new TemplateInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.description = source["description"];
	        this.description_en = source["description_en"];
	        this.version = source["version"];
	        this.user = source["user"];
	        this.module = source["module"];
	        this.plugins = source["plugins"];
	    }
	}
	export class TemplateRecommendation {
	    template: string;
	    name: string;
	    description: string;
	    match: number;
	    tags: string[];
	    provider: string;
	    version: string;
	    installed: boolean;
	
	    static createFrom(source: any = {}) {
	        return new TemplateRecommendation(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.template = source["template"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.match = source["match"];
	        this.tags = source["tags"];
	        this.provider = source["provider"];
	        this.version = source["version"];
	        this.installed = source["installed"];
	    }
	}
	export class TemplateUpdateInfo {
	    name: string;
	    localVersion: string;
	    latestVersion: string;
	    hasUpdate: boolean;
	
	    static createFrom(source: any = {}) {
	        return new TemplateUpdateInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.localVersion = source["localVersion"];
	        this.latestVersion = source["latestVersion"];
	        this.hasUpdate = source["hasUpdate"];
	    }
	}
	export class TemplateValidateDiagnostic {
	    severity: string;
	    summary: string;
	    detail: string;
	    filename: string;
	    line: number;
	
	    static createFrom(source: any = {}) {
	        return new TemplateValidateDiagnostic(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.severity = source["severity"];
	        this.summary = source["summary"];
	        this.detail = source["detail"];
	        this.filename = source["filename"];
	        this.line = source["line"];
	    }
	}
	export class TemplateValidateResult {
	    valid: boolean;
	    error_count: number;
	    warning_count: number;
	    diagnostics: TemplateValidateDiagnostic[];
	
	    static createFrom(source: any = {}) {
	        return new TemplateValidateResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.valid = source["valid"];
	        this.error_count = source["error_count"];
	        this.warning_count = source["warning_count"];
	        this.diagnostics = this.convertValues(source["diagnostics"], TemplateValidateDiagnostic);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TemplateVariableValidation {
	    allowed_values?: string[];
	
	    static createFrom(source: any = {}) {
	        return new TemplateVariableValidation(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.allowed_values = source["allowed_values"];
	    }
	}
	export class TemplateVariable {
	    name: string;
	    type: string;
	    description: string;
	    defaultValue: string;
	    required: boolean;
	    sensitive: boolean;
	    validation?: TemplateVariableValidation;
	
	    static createFrom(source: any = {}) {
	        return new TemplateVariable(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.type = source["type"];
	        this.description = source["description"];
	        this.defaultValue = source["defaultValue"];
	        this.required = source["required"];
	        this.sensitive = source["sensitive"];
	        this.validation = this.convertValues(source["validation"], TemplateVariableValidation);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class TerraformMirrorConfig {
	    enabled: boolean;
	    configPath: string;
	    managed: boolean;
	    fromEnv: boolean;
	    providers: string[];
	
	    static createFrom(source: any = {}) {
	        return new TerraformMirrorConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.configPath = source["configPath"];
	        this.managed = source["managed"];
	        this.fromEnv = source["fromEnv"];
	        this.providers = source["providers"];
	    }
	}
	export class VersionCheckResult {
	    currentVersion: string;
	    latestVersion: string;
	    hasUpdate: boolean;
	    downloadURL: string;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new VersionCheckResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.currentVersion = source["currentVersion"];
	        this.latestVersion = source["latestVersion"];
	        this.hasUpdate = source["hasUpdate"];
	        this.downloadURL = source["downloadURL"];
	        this.error = source["error"];
	    }
	}
	export class UpdateCheckResult {
	    redc: VersionCheckResult;
	    templates: TemplateUpdateInfo[];
	    plugins: PluginUpdateInfo[];
	
	    static createFrom(source: any = {}) {
	        return new UpdateCheckResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.redc = this.convertValues(source["redc"], VersionCheckResult);
	        this.templates = this.convertValues(source["templates"], TemplateUpdateInfo);
	        this.plugins = this.convertValues(source["plugins"], PluginUpdateInfo);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class WebhookConfig {
	    enabled: boolean;
	    slack: string;
	    dingtalk: string;
	    dingtalkSecret: string;
	    feishu: string;
	    feishuSecret: string;
	    discord: string;
	    wecom: string;
	
	    static createFrom(source: any = {}) {
	        return new WebhookConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.slack = source["slack"];
	        this.dingtalk = source["dingtalk"];
	        this.dingtalkSecret = source["dingtalkSecret"];
	        this.feishu = source["feishu"];
	        this.feishuSecret = source["feishuSecret"];
	        this.discord = source["discord"];
	        this.wecom = source["wecom"];
	    }
	}

}

export namespace mod {
	
	export class FallbackProvider {
	    name: string;
	    provider: string;
	    apiKey: string;
	    baseUrl: string;
	    model: string;
	
	    static createFrom(source: any = {}) {
	        return new FallbackProvider(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.provider = source["provider"];
	        this.apiKey = source["apiKey"];
	        this.baseUrl = source["baseUrl"];
	        this.model = source["model"];
	    }
	}
	export class AIConfig {
	    provider: string;
	    apiKey?: string;
	    baseUrl: string;
	    model: string;
	    maxToolRounds?: number;
	    contextWindow?: number;
	    enableAskUser?: boolean;
	    fallbackProviders?: FallbackProvider[];
	
	    static createFrom(source: any = {}) {
	        return new AIConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.provider = source["provider"];
	        this.apiKey = source["apiKey"];
	        this.baseUrl = source["baseUrl"];
	        this.model = source["model"];
	        this.maxToolRounds = source["maxToolRounds"];
	        this.contextWindow = source["contextWindow"];
	        this.enableAskUser = source["enableAskUser"];
	        this.fallbackProviders = this.convertValues(source["fallbackProviders"], FallbackProvider);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AuditLogEntry {
	    id: number;
	    timestamp: string;
	    username: string;
	    role: string;
	    method: string;
	    args: string;
	    success: boolean;
	    error?: string;
	    ip?: string;
	
	    static createFrom(source: any = {}) {
	        return new AuditLogEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.timestamp = source["timestamp"];
	        this.username = source["username"];
	        this.role = source["role"];
	        this.method = source["method"];
	        this.args = source["args"];
	        this.success = source["success"];
	        this.error = source["error"];
	        this.ip = source["ip"];
	    }
	}
	export class VariableValidation {
	    pattern?: string;
	    min_length?: number;
	    max_length?: number;
	    allowed_values?: string[];
	
	    static createFrom(source: any = {}) {
	        return new VariableValidation(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pattern = source["pattern"];
	        this.min_length = source["min_length"];
	        this.max_length = source["max_length"];
	        this.allowed_values = source["allowed_values"];
	    }
	}
	export class TemplateVariable {
	    name: string;
	    type: string;
	    description: string;
	    required: boolean;
	    default_value?: string;
	    validation?: VariableValidation;
	
	    static createFrom(source: any = {}) {
	        return new TemplateVariable(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.type = source["type"];
	        this.description = source["description"];
	        this.required = source["required"];
	        this.default_value = source["default_value"];
	        this.validation = this.convertValues(source["validation"], VariableValidation);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class BaseTemplate {
	    name: string;
	    description: string;
	    description_en: string;
	    version: string;
	    variables: TemplateVariable[];
	    provider: string;
	    providers: string[];
	    template: string;
	    user: string;
	    redc_module?: string;
	
	    static createFrom(source: any = {}) {
	        return new BaseTemplate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.description = source["description"];
	        this.description_en = source["description_en"];
	        this.version = source["version"];
	        this.variables = this.convertValues(source["variables"], TemplateVariable);
	        this.provider = source["provider"];
	        this.providers = source["providers"];
	        this.template = source["template"];
	        this.user = source["user"];
	        this.redc_module = source["redc_module"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class BatchOperationResult {
	    deployment_id: string;
	    success: boolean;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new BatchOperationResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.deployment_id = source["deployment_id"];
	        this.success = source["success"];
	        this.error = source["error"];
	    }
	}
	export class ComposeTemplate {
	    name: string;
	    nameZh: string;
	    type: string;
	    category: string;
	    description?: string;
	    user?: string;
	    version?: string;
	    composeFile: string;
	    path: string;
	
	    static createFrom(source: any = {}) {
	        return new ComposeTemplate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.nameZh = source["nameZh"];
	        this.type = source["type"];
	        this.category = source["category"];
	        this.description = source["description"];
	        this.user = source["user"];
	        this.version = source["version"];
	        this.composeFile = source["composeFile"];
	        this.path = source["path"];
	    }
	}
	export class CostEstimate {
	    monthly_cost: number;
	    currency: string;
	    details: Record<string, number>;
	
	    static createFrom(source: any = {}) {
	        return new CostEstimate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.monthly_cost = source["monthly_cost"];
	        this.currency = source["currency"];
	        this.details = source["details"];
	    }
	}
	export class DeploymentConfig {
	    name: string;
	    template_name: string;
	    provider: string;
	    region: string;
	    instance_type: string;
	    userdata?: string;
	    is_spot_instance: boolean;
	    variables: Record<string, string>;
	    created_at: time.Time;
	    updated_at: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new DeploymentConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.template_name = source["template_name"];
	        this.provider = source["provider"];
	        this.region = source["region"];
	        this.instance_type = source["instance_type"];
	        this.userdata = source["userdata"];
	        this.is_spot_instance = source["is_spot_instance"];
	        this.variables = source["variables"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CustomDeployment {
	    id: string;
	    name: string;
	    template_name: string;
	    config?: DeploymentConfig;
	    state: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    outputs?: Record<string, any>;
	
	    static createFrom(source: any = {}) {
	        return new CustomDeployment(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.template_name = source["template_name"];
	        this.config = this.convertValues(source["config"], DeploymentConfig);
	        this.state = source["state"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.outputs = source["outputs"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DeploymentChangeHistory {
	    id: string;
	    deployment_id: string;
	    change_type: string;
	    old_value?: Record<string, any>;
	    new_value?: Record<string, any>;
	    operator?: string;
	    timestamp: time.Time;
	    description?: string;
	
	    static createFrom(source: any = {}) {
	        return new DeploymentChangeHistory(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.deployment_id = source["deployment_id"];
	        this.change_type = source["change_type"];
	        this.old_value = source["old_value"];
	        this.new_value = source["new_value"];
	        this.operator = source["operator"];
	        this.timestamp = this.convertValues(source["timestamp"], time.Time);
	        this.description = source["description"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class F8xCategoryInfo {
	    id: string;
	    name: string;
	    nameZh: string;
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new F8xCategoryInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.nameZh = source["nameZh"];
	        this.count = source["count"];
	    }
	}
	export class F8xSubTool {
	    name: string;
	    url?: string;
	
	    static createFrom(source: any = {}) {
	        return new F8xSubTool(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.url = source["url"];
	    }
	}
	export class F8xModule {
	    id: string;
	    name: string;
	    nameZh: string;
	    flag: string;
	    category: string;
	    description: string;
	    descriptionZh: string;
	    tags: string[];
	    includes?: F8xSubTool[];
	
	    static createFrom(source: any = {}) {
	        return new F8xModule(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.nameZh = source["nameZh"];
	        this.flag = source["flag"];
	        this.category = source["category"];
	        this.description = source["description"];
	        this.descriptionZh = source["descriptionZh"];
	        this.tags = source["tags"];
	        this.includes = this.convertValues(source["includes"], F8xSubTool);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class F8xPreset {
	    id: string;
	    name: string;
	    nameZh: string;
	    description: string;
	    flags: string[];
	
	    static createFrom(source: any = {}) {
	        return new F8xPreset(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.nameZh = source["nameZh"];
	        this.description = source["description"];
	        this.flags = source["flags"];
	    }
	}
	
	
	export class HTTPUser {
	    username: string;
	    token: string;
	    role: string;
	
	    static createFrom(source: any = {}) {
	        return new HTTPUser(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.username = source["username"];
	        this.token = source["token"];
	        this.role = source["role"];
	    }
	}
	export class InstanceType {
	    code: string;
	    name: string;
	    cpu: number;
	    memory: number;
	    description: string;
	    price?: number;
	
	    static createFrom(source: any = {}) {
	        return new InstanceType(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	        this.name = source["name"];
	        this.cpu = source["cpu"];
	        this.memory = source["memory"];
	        this.description = source["description"];
	        this.price = source["price"];
	    }
	}
	export class ProfileInfo {
	    id: string;
	    name: string;
	    configPath: string;
	    templateDir: string;
	    aiConfig?: AIConfig;
	
	    static createFrom(source: any = {}) {
	        return new ProfileInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.configPath = source["configPath"];
	        this.templateDir = source["templateDir"];
	        this.aiConfig = this.convertValues(source["aiConfig"], AIConfig);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Region {
	    code: string;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new Region(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	        this.name = source["name"];
	    }
	}
	export class ScheduledTask {
	    id: string;
	    caseId: string;
	    caseName: string;
	    action: string;
	    scheduledAt: time.Time;
	    createdAt: time.Time;
	    status: string;
	    error?: string;
	    repeatType?: string;
	    repeatInterval?: number;
	    completedAt?: time.Time;
	    sshCommand?: string;
	    taskResult?: string;
	    notifyEnabled?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ScheduledTask(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.caseId = source["caseId"];
	        this.caseName = source["caseName"];
	        this.action = source["action"];
	        this.scheduledAt = this.convertValues(source["scheduledAt"], time.Time);
	        this.createdAt = this.convertValues(source["createdAt"], time.Time);
	        this.status = source["status"];
	        this.error = source["error"];
	        this.repeatType = source["repeatType"];
	        this.repeatInterval = source["repeatInterval"];
	        this.completedAt = this.convertValues(source["completedAt"], time.Time);
	        this.sshCommand = source["sshCommand"];
	        this.taskResult = source["taskResult"];
	        this.notifyEnabled = source["notifyEnabled"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class TimelineEvent {
	    id: number;
	    timestamp: string;
	    category: string;
	    eventType: string;
	    caseId: string;
	    caseName: string;
	    message: string;
	    detail: string;
	    level: string;
	
	    static createFrom(source: any = {}) {
	        return new TimelineEvent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.timestamp = source["timestamp"];
	        this.category = source["category"];
	        this.eventType = source["eventType"];
	        this.caseId = source["caseId"];
	        this.caseName = source["caseName"];
	        this.message = source["message"];
	        this.detail = source["detail"];
	        this.level = source["level"];
	    }
	}
	export class TimelineListResult {
	    events: TimelineEvent[];
	    total: number;
	
	    static createFrom(source: any = {}) {
	        return new TimelineListResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.events = this.convertValues(source["events"], TimelineEvent);
	        this.total = source["total"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class UserdataTemplate {
	    name: string;
	    nameZh: string;
	    type: string;
	    category: string;
	    url?: string;
	    description?: string;
	    installNotes?: string;
	    minMemoryMB?: number;
	    script: string;
	
	    static createFrom(source: any = {}) {
	        return new UserdataTemplate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.nameZh = source["nameZh"];
	        this.type = source["type"];
	        this.category = source["category"];
	        this.url = source["url"];
	        this.description = source["description"];
	        this.installNotes = source["installNotes"];
	        this.minMemoryMB = source["minMemoryMB"];
	        this.script = source["script"];
	    }
	}
	export class ValidationError {
	    field: string;
	    message: string;
	    code: string;
	
	    static createFrom(source: any = {}) {
	        return new ValidationError(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.field = source["field"];
	        this.message = source["message"];
	        this.code = source["code"];
	    }
	}
	export class ValidationWarning {
	    field: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new ValidationWarning(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.field = source["field"];
	        this.message = source["message"];
	    }
	}
	export class ValidationResult {
	    valid: boolean;
	    errors?: ValidationError[];
	    warnings?: ValidationWarning[];
	
	    static createFrom(source: any = {}) {
	        return new ValidationResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.valid = source["valid"];
	        this.errors = this.convertValues(source["errors"], ValidationError);
	        this.warnings = this.convertValues(source["warnings"], ValidationWarning);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	

}

export namespace plugin {
	
	export class ConfigField {
	    type: string;
	    required?: boolean;
	    description?: string;
	    default?: string;
	
	    static createFrom(source: any = {}) {
	        return new ConfigField(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.required = source["required"];
	        this.description = source["description"];
	        this.default = source["default"];
	    }
	}
	export class RegistryPlugin {
	    name: string;
	    version: string;
	    description: string;
	    description_en?: string;
	    author?: string;
	    category?: string;
	    tags?: string[];
	    min_redc_version?: string;
	    url: string;
	
	    static createFrom(source: any = {}) {
	        return new RegistryPlugin(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.version = source["version"];
	        this.description = source["description"];
	        this.description_en = source["description_en"];
	        this.author = source["author"];
	        this.category = source["category"];
	        this.tags = source["tags"];
	        this.min_redc_version = source["min_redc_version"];
	        this.url = source["url"];
	    }
	}

}

export namespace sshutil {
	
	export class FileInfo {
	    name: string;
	    size: number;
	    mode: string;
	    modTime: time.Time;
	    isDir: boolean;
	    isLink: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FileInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.size = source["size"];
	        this.mode = source["mode"];
	        this.modTime = this.convertValues(source["modTime"], time.Time);
	        this.isDir = source["isDir"];
	        this.isLink = source["isLink"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace time {
	
	export class Time {
	
	
	    static createFrom(source: any = {}) {
	        return new Time(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}

}

