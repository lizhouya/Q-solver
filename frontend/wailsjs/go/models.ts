export namespace app {

	export class MobileInfo {
	    available: boolean;
	    url: string;
	    port: number;
	    status: string;
	    retryable: boolean;

	    static createFrom(source: any = {}) {
	        return new MobileInfo(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.available = source["available"];
	        this.url = source["url"];
	        this.port = source["port"];
	        this.status = source["status"];
	        this.retryable = source["retryable"];
	    }
	}
	export class QuestionBankExportDraft {
	    sessionId: string;
	    company: string;
	    type: string;
	    name: string;
	    directory: string;
	    autoOrganize: boolean;
	    numberedPrefix: string;
	    count: number;
	    startedAt: string;

	    static createFrom(source: any = {}) {
	        return new QuestionBankExportDraft(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sessionId = source["sessionId"];
	        this.company = source["company"];
	        this.type = source["type"];
	        this.name = source["name"];
	        this.directory = source["directory"];
	        this.autoOrganize = source["autoOrganize"];
	        this.numberedPrefix = source["numberedPrefix"];
	        this.count = source["count"];
	        this.startedAt = source["startedAt"];
	    }
	}
	export class QuestionBankExportRequest {
	    sessionId: string;
	    company: string;
	    type: string;
	    name: string;
	    directory: string;
	    autoOrganize: boolean;

	    static createFrom(source: any = {}) {
	        return new QuestionBankExportRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sessionId = source["sessionId"];
	        this.company = source["company"];
	        this.type = source["type"];
	        this.name = source["name"];
	        this.directory = source["directory"];
	        this.autoOrganize = source["autoOrganize"];
	    }
	}
	export class QuestionBankPreferences {
	    version: number;
	    archiveDir: string;
	    company: string;
	    autoOrganize: boolean;

	    static createFrom(source: any = {}) {
	        return new QuestionBankPreferences(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.archiveDir = source["archiveDir"];
	        this.company = source["company"];
	        this.autoOrganize = source["autoOrganize"];
	    }
	}
	export class QuestionBankStatus {
	    root: string;
	    preferences: QuestionBankPreferences;
	    session?: questionbank.SessionInfo;
	    unsaved: boolean;
	    busy: boolean;
	    organizing: boolean;
	    operation: string;
	    progress: questionbank.Progress;
	    lastExport: questionbank.ExportResult;
	    message: string;

	    static createFrom(source: any = {}) {
	        return new QuestionBankStatus(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.root = source["root"];
	        this.preferences = this.convertValues(source["preferences"], QuestionBankPreferences);
	        this.session = this.convertValues(source["session"], questionbank.SessionInfo);
	        this.unsaved = source["unsaved"];
	        this.busy = source["busy"];
	        this.organizing = source["organizing"];
	        this.operation = source["operation"];
	        this.progress = this.convertValues(source["progress"], questionbank.Progress);
	        this.lastExport = this.convertValues(source["lastExport"], questionbank.ExportResult);
	        this.message = source["message"];
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

export namespace config {

	export class Config {
	    apiKey?: string;
	    baseURL?: string;
	    model?: string;
	    prompt?: string;
	    domainId?: string;
	    opacity?: number;
	    noCompression?: boolean;
	    compressionQuality?: number;
	    sharpening?: number;
	    grayscale?: boolean;
	    keepContext?: boolean;
	    interruptThinking?: boolean;
	    screenshotMode?: string;
	    outputMode?: string;
	    resumePath?: string;
	    resumeContent?: string;
	    shortcuts?: Record<string, shortcut.KeyBinding>;
	    assistantModel?: string;
	    windowWidth?: number;
	    windowHeight?: number;
	    theme?: string;

	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.apiKey = source["apiKey"];
	        this.baseURL = source["baseURL"];
	        this.model = source["model"];
	        this.prompt = source["prompt"];
	        this.domainId = source["domainId"];
	        this.opacity = source["opacity"];
	        this.noCompression = source["noCompression"];
	        this.compressionQuality = source["compressionQuality"];
	        this.sharpening = source["sharpening"];
	        this.grayscale = source["grayscale"];
	        this.keepContext = source["keepContext"];
	        this.interruptThinking = source["interruptThinking"];
	        this.screenshotMode = source["screenshotMode"];
	        this.outputMode = source["outputMode"];
	        this.resumePath = source["resumePath"];
	        this.resumeContent = source["resumeContent"];
	        this.shortcuts = this.convertValues(source["shortcuts"], shortcut.KeyBinding, true);
	        this.assistantModel = source["assistantModel"];
	        this.windowWidth = source["windowWidth"];
	        this.windowHeight = source["windowHeight"];
	        this.theme = source["theme"];
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

export namespace domain {

	export class DomainItem {
	    id: string;
	    label: string;
	    icon: string;
	    description: string;

	    static createFrom(source: any = {}) {
	        return new DomainItem(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.icon = source["icon"];
	        this.description = source["description"];
	    }
	}
	export class Category {
	    id: string;
	    label: string;
	    items: DomainItem[];

	    static createFrom(source: any = {}) {
	        return new Category(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.items = this.convertValues(source["items"], DomainItem);
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

export namespace questionbank {

	export class ExportOptions {
	    groupBy: string;
	    name: string;
	    company?: string;
	    type?: string;
	    directory?: string;
	    rawOnly?: boolean;

	    static createFrom(source: any = {}) {
	        return new ExportOptions(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.groupBy = source["groupBy"];
	        this.name = source["name"];
	        this.company = source["company"];
	        this.type = source["type"];
	        this.directory = source["directory"];
	        this.rawOnly = source["rawOnly"];
	    }
	}
	export class ExportResult {
	    directory: string;
	    markdown: string;
	    rawMarkdown: string;
	    records: number;
	    questions: number;

	    static createFrom(source: any = {}) {
	        return new ExportResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.directory = source["directory"];
	        this.markdown = source["markdown"];
	        this.rawMarkdown = source["rawMarkdown"];
	        this.records = source["records"];
	        this.questions = source["questions"];
	    }
	}
	export class Progress {
	    done: number;
	    total: number;
	    failed: number;

	    static createFrom(source: any = {}) {
	        return new Progress(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.done = source["done"];
	        this.total = source["total"];
	        this.failed = source["failed"];
	    }
	}
	export class SessionInfo {
	    id: string;
	    company: string;
	    type: string;
	    state: string;
	    count: number;
	    startedAt: string;
	    exportDirectory?: string;

	    static createFrom(source: any = {}) {
	        return new SessionInfo(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.company = source["company"];
	        this.type = source["type"];
	        this.state = source["state"];
	        this.count = source["count"];
	        this.startedAt = source["startedAt"];
	        this.exportDirectory = source["exportDirectory"];
	    }
	}

}

export namespace screen {

	export class PreviewResult {
	    imgBytes: number[];
	    base64: string;
	    size: string;

	    static createFrom(source: any = {}) {
	        return new PreviewResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.imgBytes = source["imgBytes"];
	        this.base64 = source["base64"];
	        this.size = source["size"];
	    }
	}

}

export namespace shortcut {

	export class KeyBinding {
	    vkCode: string;
	    keyName: string;

	    static createFrom(source: any = {}) {
	        return new KeyBinding(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.vkCode = source["vkCode"];
	        this.keyName = source["keyName"];
	    }
	}

}

