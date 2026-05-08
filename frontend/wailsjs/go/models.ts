export namespace character {
	
	export class CharacterSummary {
	    id: string;
	    name: string;
	    gender: string;
	    race: string;
	    role: string;
	    personality: string;
	    status: string;
	    tags: string;
	    avatar: string;
	    created_at: string;
	
	    static createFrom(source: any = {}) {
	        return new CharacterSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.gender = source["gender"];
	        this.race = source["race"];
	        this.role = source["role"];
	        this.personality = source["personality"];
	        this.status = source["status"];
	        this.tags = source["tags"];
	        this.avatar = source["avatar"];
	        this.created_at = source["created_at"];
	    }
	}

}

export namespace config {
	
	export class EmbeddingConfig {
	    engine: string;
	    server_path: string;
	    model_path: string;
	    port: number;
	
	    static createFrom(source: any = {}) {
	        return new EmbeddingConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.engine = source["engine"];
	        this.server_path = source["server_path"];
	        this.model_path = source["model_path"];
	        this.port = source["port"];
	    }
	}
	export class VectorDBConfig {
	    dimension: number;
	
	    static createFrom(source: any = {}) {
	        return new VectorDBConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.dimension = source["dimension"];
	    }
	}
	export class LLMConfig {
	    provider: string;
	    api_key: string;
	    base_url: string;
	    chat_model: string;
	
	    static createFrom(source: any = {}) {
	        return new LLMConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.provider = source["provider"];
	        this.api_key = source["api_key"];
	        this.base_url = source["base_url"];
	        this.chat_model = source["chat_model"];
	    }
	}
	export class Config {
	    llm: LLMConfig;
	    vector_db: VectorDBConfig;
	    embedding: EmbeddingConfig;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.llm = this.convertValues(source["llm"], LLMConfig);
	        this.vector_db = this.convertValues(source["vector_db"], VectorDBConfig);
	        this.embedding = this.convertValues(source["embedding"], EmbeddingConfig);
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

export namespace coordinator {
	
	export class SessionEvent {
	    id: string;
	    type: string;
	    agent: string;
	    content: string;
	    user_action: string;
	    // Go type: time
	    timestamp: any;
	
	    static createFrom(source: any = {}) {
	        return new SessionEvent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.type = source["type"];
	        this.agent = source["agent"];
	        this.content = source["content"];
	        this.user_action = source["user_action"];
	        this.timestamp = this.convertValues(source["timestamp"], null);
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

export namespace foreshadow {
	
	export class CandidateForeshadow {
	    text: string;
	    type: string;
	    confidence: number;
	    reason: string;
	    start_index: number;
	    end_index: number;
	
	    static createFrom(source: any = {}) {
	        return new CandidateForeshadow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.text = source["text"];
	        this.type = source["type"];
	        this.confidence = source["confidence"];
	        this.reason = source["reason"];
	        this.start_index = source["start_index"];
	        this.end_index = source["end_index"];
	    }
	}
	export class ForeshadowSummary {
	    id: string;
	    description: string;
	    type: string;
	    status: string;
	    chapter_planted: string;
	    planned_reveal_chapter: number;
	    reveal_progress: number;
	    priority: string;
	    confidence: number;
	    created_at: string;
	
	    static createFrom(source: any = {}) {
	        return new ForeshadowSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.description = source["description"];
	        this.type = source["type"];
	        this.status = source["status"];
	        this.chapter_planted = source["chapter_planted"];
	        this.planned_reveal_chapter = source["planned_reveal_chapter"];
	        this.reveal_progress = source["reveal_progress"];
	        this.priority = source["priority"];
	        this.confidence = source["confidence"];
	        this.created_at = source["created_at"];
	    }
	}
	export class HealthReport {
	    total: number;
	    planted: number;
	    partially_revealed: number;
	    revealed: number;
	    avg_reveal_gap: number;
	    stale_count: number;
	    stale_warnings: string[];
	    priority_counts: Record<string, number>;
	
	    static createFrom(source: any = {}) {
	        return new HealthReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total = source["total"];
	        this.planted = source["planted"];
	        this.partially_revealed = source["partially_revealed"];
	        this.revealed = source["revealed"];
	        this.avg_reveal_gap = source["avg_reveal_gap"];
	        this.stale_count = source["stale_count"];
	        this.stale_warnings = source["stale_warnings"];
	        this.priority_counts = source["priority_counts"];
	    }
	}
	export class RevealPlan {
	    method: string;
	    paragraph: string;
	
	    static createFrom(source: any = {}) {
	        return new RevealPlan(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.method = source["method"];
	        this.paragraph = source["paragraph"];
	    }
	}
	export class RevealSuggestion {
	    foreshadow_id: string;
	    description: string;
	    foreshadow_type: string;
	    current_status: string;
	    plans: RevealPlan[];
	
	    static createFrom(source: any = {}) {
	        return new RevealSuggestion(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.foreshadow_id = source["foreshadow_id"];
	        this.description = source["description"];
	        this.foreshadow_type = source["foreshadow_type"];
	        this.current_status = source["current_status"];
	        this.plans = this.convertValues(source["plans"], RevealPlan);
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

export namespace inspiration {
	
	export class InspirationMatch {
	    id: string;
	    content: string;
	    tags: string[];
	    score: number;
	    match_type: string;
	
	    static createFrom(source: any = {}) {
	        return new InspirationMatch(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.content = source["content"];
	        this.tags = source["tags"];
	        this.score = source["score"];
	        this.match_type = source["match_type"];
	    }
	}
	export class InspirationSummary {
	    id: string;
	    content: string;
	    tags: string[];
	    source: string;
	    created_at: string;
	
	    static createFrom(source: any = {}) {
	        return new InspirationSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.content = source["content"];
	        this.tags = source["tags"];
	        this.source = source["source"];
	        this.created_at = source["created_at"];
	    }
	}

}

export namespace models {
	
	export class Chapter {
	    id: string;
	    title: string;
	    content: string;
	    sort_order: number;
	    novel_id: string;
	    volume_id: string;
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    updated_at: any;
	
	    static createFrom(source: any = {}) {
	        return new Chapter(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.content = source["content"];
	        this.sort_order = source["sort_order"];
	        this.novel_id = source["novel_id"];
	        this.volume_id = source["volume_id"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.updated_at = this.convertValues(source["updated_at"], null);
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
	export class ChapterSummary {
	    id: string;
	    title: string;
	    sort_order: number;
	    volume_id: string;
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    updated_at: any;
	
	    static createFrom(source: any = {}) {
	        return new ChapterSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.sort_order = source["sort_order"];
	        this.volume_id = source["volume_id"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.updated_at = this.convertValues(source["updated_at"], null);
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
	export class Character {
	    id: string;
	    name: string;
	    gender: string;
	    race: string;
	    personality: string;
	    description: string;
	    first_chapter: string;
	    tags: string;
	    avatar: string;
	    role: string;
	    status: string;
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    updated_at: any;
	
	    static createFrom(source: any = {}) {
	        return new Character(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.gender = source["gender"];
	        this.race = source["race"];
	        this.personality = source["personality"];
	        this.description = source["description"];
	        this.first_chapter = source["first_chapter"];
	        this.tags = source["tags"];
	        this.avatar = source["avatar"];
	        this.role = source["role"];
	        this.status = source["status"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.updated_at = this.convertValues(source["updated_at"], null);
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
	export class Foreshadowing {
	    id: string;
	    description: string;
	    type: string;
	    status: string;
	    chapter_planted: string;
	    planned_reveal_chapter: number;
	    reveal_progress: number;
	    reveal_plan: string;
	    priority: string;
	    confidence: number;
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    updated_at: any;
	
	    static createFrom(source: any = {}) {
	        return new Foreshadowing(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.description = source["description"];
	        this.type = source["type"];
	        this.status = source["status"];
	        this.chapter_planted = source["chapter_planted"];
	        this.planned_reveal_chapter = source["planned_reveal_chapter"];
	        this.reveal_progress = source["reveal_progress"];
	        this.reveal_plan = source["reveal_plan"];
	        this.priority = source["priority"];
	        this.confidence = source["confidence"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.updated_at = this.convertValues(source["updated_at"], null);
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
	export class StyleProfile {
	    id: string;
	    name: string;
	    features: string;
	    samples: string;
	    description: string;
	    source_chapters: string;
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    updated_at: any;
	
	    static createFrom(source: any = {}) {
	        return new StyleProfile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.features = source["features"];
	        this.samples = source["samples"];
	        this.description = source["description"];
	        this.source_chapters = source["source_chapters"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.updated_at = this.convertValues(source["updated_at"], null);
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
	export class VolumeSummary {
	    id: string;
	    name: string;
	    sort_order: number;
	
	    static createFrom(source: any = {}) {
	        return new VolumeSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.sort_order = source["sort_order"];
	    }
	}
	export class WorldSetting {
	    id: string;
	    title: string;
	    content: string;
	    type: string;
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    updated_at: any;
	
	    static createFrom(source: any = {}) {
	        return new WorldSetting(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.content = source["content"];
	        this.type = source["type"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.updated_at = this.convertValues(source["updated_at"], null);
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

export namespace plotengine {
	
	export class RevealDetail {
	    foreshadow_id: string;
	    foreshadow_desc: string;
	    how_revealed: string;
	
	    static createFrom(source: any = {}) {
	        return new RevealDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.foreshadow_id = source["foreshadow_id"];
	        this.foreshadow_desc = source["foreshadow_desc"];
	        this.how_revealed = source["how_revealed"];
	    }
	}
	export class PlotPoint {
	    description: string;
	    order: number;
	    type: string;
	
	    static createFrom(source: any = {}) {
	        return new PlotPoint(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.description = source["description"];
	        this.order = source["order"];
	        this.type = source["type"];
	    }
	}
	export class Branch {
	    id: string;
	    title: string;
	    summary: string;
	    plot_points: PlotPoint[];
	    reveal_details: RevealDetail[];
	
	    static createFrom(source: any = {}) {
	        return new Branch(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.summary = source["summary"];
	        this.plot_points = this.convertValues(source["plot_points"], PlotPoint);
	        this.reveal_details = this.convertValues(source["reveal_details"], RevealDetail);
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
	export class ExpectationPoint {
	    position: string;
	    score: number;
	
	    static createFrom(source: any = {}) {
	        return new ExpectationPoint(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.position = source["position"];
	        this.score = source["score"];
	    }
	}
	export class BranchAnalysis {
	    branch_id: string;
	    logic_issues: string[];
	    rhythm_notes: string;
	    expectation_curve: ExpectationPoint[];
	    reader_tags: Record<string, Array<string>>;
	    overall_rating: number;
	
	    static createFrom(source: any = {}) {
	        return new BranchAnalysis(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.branch_id = source["branch_id"];
	        this.logic_issues = source["logic_issues"];
	        this.rhythm_notes = source["rhythm_notes"];
	        this.expectation_curve = this.convertValues(source["expectation_curve"], ExpectationPoint);
	        this.reader_tags = source["reader_tags"];
	        this.overall_rating = source["overall_rating"];
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
	
	export class GenerationParams {
	    chapter_summary: string;
	    branch_count: number;
	    foreshadow_ids: string[];
	
	    static createFrom(source: any = {}) {
	        return new GenerationParams(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.chapter_summary = source["chapter_summary"];
	        this.branch_count = source["branch_count"];
	        this.foreshadow_ids = source["foreshadow_ids"];
	    }
	}
	

}

export namespace setting {
	
	export class ConflictWarning {
	    conflict_desc: string;
	    suggested_fix: string;
	    reference_text: string;
	    setting_title: string;
	
	    static createFrom(source: any = {}) {
	        return new ConflictWarning(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.conflict_desc = source["conflict_desc"];
	        this.suggested_fix = source["suggested_fix"];
	        this.reference_text = source["reference_text"];
	        this.setting_title = source["setting_title"];
	    }
	}
	export class SettingInfo {
	    id: string;
	    title: string;
	    type: string;
	
	    static createFrom(source: any = {}) {
	        return new SettingInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.type = source["type"];
	    }
	}

}

export namespace style {
	
	export class StyleProfileSummary {
	    id: string;
	    name: string;
	    description: string;
	    created_at: string;
	
	    static createFrom(source: any = {}) {
	        return new StyleProfileSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.created_at = source["created_at"];
	    }
	}

}

