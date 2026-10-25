export namespace answersummary {

	export class Item {
	    label: string;
	    kind: string;
	    value: string;
	    status: string;

	    static createFrom(source: any = {}) {
	        return new Item(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.label = source["label"];
	        this.kind = source["kind"];
	        this.value = source["value"];
	        this.status = source["status"];
	    }
	}

}

export namespace app {

	export class MobileInfo {
	    available: boolean;
	    url: string;
	    port: number;
	    status: string;
	    retryable: boolean;
	    stableLinkEnabled: boolean;
	    canClear: boolean;
	    canResetLink: boolean;

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
	        this.stableLinkEnabled = source["stableLinkEnabled"];
	        this.canClear = source["canClear"];
	        this.canResetLink = source["canResetLink"];
	    }
	}
	export class MobileCommandResult {
	    success: boolean;
	    code: string;
	    info: MobileInfo;

	    static createFrom(source: any = {}) {
	        return new MobileCommandResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.code = source["code"];
	        this.info = this.convertValues(source["info"], MobileInfo);
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

	export class QuestionBankImageInfo {
	    index: number;
	    mime: string;

	    static createFrom(source: any = {}) {
	        return new QuestionBankImageInfo(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.index = source["index"];
	        this.mime = source["mime"];
	    }
	}
	export class QuestionBankRecordDetail {
	    answerItems?: answersummary.Item[];
	    answerCopyText?: string;
	    id: string;
	    createdAt: string;
	    model: string;
	    scene: string;
	    answer: string;
	    status: string;
	    organizeError?: string;
	    metadataSuggestion?: questionbank.SourceMetadataSuggestion;
	    images: QuestionBankImageInfo[];
	    questions?: questionbank.Question[];
	    keepContext?: boolean;

	    static createFrom(source: any = {}) {
	        return new QuestionBankRecordDetail(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.answerItems = this.convertValues(source["answerItems"], answersummary.Item);
	        this.answerCopyText = source["answerCopyText"];
	        this.id = source["id"];
	        this.createdAt = source["createdAt"];
	        this.model = source["model"];
	        this.scene = source["scene"];
	        this.answer = source["answer"];
	        this.status = source["status"];
	        this.organizeError = source["organizeError"];
	        this.metadataSuggestion = this.convertValues(source["metadataSuggestion"], questionbank.SourceMetadataSuggestion);
	        this.images = this.convertValues(source["images"], QuestionBankImageInfo);
	        this.questions = this.convertValues(source["questions"], questionbank.Question);
	        this.keepContext = source["keepContext"];
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
	export class QuestionBankSessionDetail {
	    id: string;
	    name?: string;
	    company: string;
	    issuer?: string;
	    type: string;
	    nameCustomized?: boolean;
	    state: string;
	    count: number;
	    startedAt: string;
	    endedAt?: string;
	    exportedAt?: string;
	    discardedAt?: string;
	    records: QuestionBankRecordDetail[];

	    static createFrom(source: any = {}) {
	        return new QuestionBankSessionDetail(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.company = source["company"];
	        this.issuer = source["issuer"];
	        this.type = source["type"];
	        this.nameCustomized = source["nameCustomized"];
	        this.state = source["state"];
	        this.count = source["count"];
	        this.startedAt = source["startedAt"];
	        this.endedAt = source["endedAt"];
	        this.exportedAt = source["exportedAt"];
	        this.discardedAt = source["discardedAt"];
	        this.records = this.convertValues(source["records"], QuestionBankRecordDetail);
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
	export class QuestionBankContinuation {
	    epoch: number;
	    contextQuality: string;
	    contextWarning?: string;
	    detail: QuestionBankSessionDetail;

	    static createFrom(source: any = {}) {
	        return new QuestionBankContinuation(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.epoch = source["epoch"];
	        this.contextQuality = source["contextQuality"];
	        this.contextWarning = source["contextWarning"];
	        this.detail = this.convertValues(source["detail"], QuestionBankSessionDetail);
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
	export class QuestionBankExportDraft {
	    sessionId: string;
	    company: string;
	    issuer: string;
	    type: string;
	    name: string;
	    nameCustomized: boolean;
	    directory: string;
	    autoOrganize: boolean;
	    numberedPrefix: string;
	    count: number;
	    startedAt: string;
	    metadataSuggestion?: questionbank.SourceMetadataSuggestion;
	    suggestionConflict?: boolean;
	    organizeReady: boolean;
	    pendingOrganize: number;
	    failedOrganize: number;

	    static createFrom(source: any = {}) {
	        return new QuestionBankExportDraft(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sessionId = source["sessionId"];
	        this.company = source["company"];
	        this.issuer = source["issuer"];
	        this.type = source["type"];
	        this.name = source["name"];
	        this.nameCustomized = source["nameCustomized"];
	        this.directory = source["directory"];
	        this.autoOrganize = source["autoOrganize"];
	        this.numberedPrefix = source["numberedPrefix"];
	        this.count = source["count"];
	        this.startedAt = source["startedAt"];
	        this.metadataSuggestion = this.convertValues(source["metadataSuggestion"], questionbank.SourceMetadataSuggestion);
	        this.suggestionConflict = source["suggestionConflict"];
	        this.organizeReady = source["organizeReady"];
	        this.pendingOrganize = source["pendingOrganize"];
	        this.failedOrganize = source["failedOrganize"];
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
	export class QuestionBankExportRequest {
	    sessionId: string;
	    company: string;
	    issuer: string;
	    type: string;
	    name: string;
	    nameCustomized: boolean;
	    directory: string;
	    autoOrganize: boolean;

	    static createFrom(source: any = {}) {
	        return new QuestionBankExportRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sessionId = source["sessionId"];
	        this.company = source["company"];
	        this.issuer = source["issuer"];
	        this.type = source["type"];
	        this.name = source["name"];
	        this.nameCustomized = source["nameCustomized"];
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


	export class QuestionBankSessionSummary {
	    id: string;
	    name?: string;
	    company: string;
	    issuer?: string;
	    type: string;
	    nameCustomized?: boolean;
	    state: string;
	    count: number;
	    startedAt: string;

	    static createFrom(source: any = {}) {
	        return new QuestionBankSessionSummary(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.company = source["company"];
	        this.issuer = source["issuer"];
	        this.type = source["type"];
	        this.nameCustomized = source["nameCustomized"];
	        this.state = source["state"];
	        this.count = source["count"];
	        this.startedAt = source["startedAt"];
	    }
	}
	export class QuestionBankSessionList {
	    sessions: QuestionBankSessionSummary[];
	    total: number;
	    hidden: number;
	    limit: number;

	    static createFrom(source: any = {}) {
	        return new QuestionBankSessionList(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sessions = this.convertValues(source["sessions"], QuestionBankSessionSummary);
	        this.total = source["total"];
	        this.hidden = source["hidden"];
	        this.limit = source["limit"];
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
	export class StartNextExamResult {
	    sessionId: string;
	    previousState: string;
	    sessionState: string;
	    sealed: boolean;
	    conversationEpoch: number;
	    clearedScreenshotCount: number;

	    static createFrom(source: any = {}) {
	        return new StartNextExamResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sessionId = source["sessionId"];
	        this.previousState = source["previousState"];
	        this.sessionState = source["sessionState"];
	        this.sealed = source["sealed"];
	        this.conversationEpoch = source["conversationEpoch"];
	        this.clearedScreenshotCount = source["clearedScreenshotCount"];
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
	    issuer?: string;
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
	        this.issuer = source["issuer"];
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
	export class Question {
	    title: string;
	    stem: string;
	    category?: string;
	    type: string;
	    options: string[];
	    knowledgePoints: string[];
	    keyPoints: string[];
	    referenceAnswer?: string;
	    imageIndexes: number[];
	    reviewNote: string;

	    static createFrom(source: any = {}) {
	        return new Question(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.stem = source["stem"];
	        this.category = source["category"];
	        this.type = source["type"];
	        this.options = source["options"];
	        this.knowledgePoints = source["knowledgePoints"];
	        this.keyPoints = source["keyPoints"];
	        this.referenceAnswer = source["referenceAnswer"];
	        this.imageIndexes = source["imageIndexes"];
	        this.reviewNote = source["reviewNote"];
	    }
	}
	export class SessionInfo {
	    id: string;
	    name?: string;
	    company: string;
	    issuer?: string;
	    type: string;
	    nameCustomized?: boolean;
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
	        this.name = source["name"];
	        this.company = source["company"];
	        this.issuer = source["issuer"];
	        this.type = source["type"];
	        this.nameCustomized = source["nameCustomized"];
	        this.state = source["state"];
	        this.count = source["count"];
	        this.startedAt = source["startedAt"];
	        this.exportDirectory = source["exportDirectory"];
	    }
	}
	export class SourceMetadataSuggestion {
	    company?: string;
	    issuer?: string;
	    type?: string;
	    reviewNote?: string;

	    static createFrom(source: any = {}) {
	        return new SourceMetadataSuggestion(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.company = source["company"];
	        this.issuer = source["issuer"];
	        this.type = source["type"];
	        this.reviewNote = source["reviewNote"];
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

