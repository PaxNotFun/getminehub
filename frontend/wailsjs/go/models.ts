export namespace config {
	
	export class Settings {
	    servers_base_dir: string;
	    max_ram_limit: number;
	    check_for_updates: boolean;
	    timeout: number;
	    notifications_enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.servers_base_dir = source["servers_base_dir"];
	        this.max_ram_limit = source["max_ram_limit"];
	        this.check_for_updates = source["check_for_updates"];
	        this.timeout = source["timeout"];
	        this.notifications_enabled = source["notifications_enabled"];
	    }
	}

}

export namespace database {
	
	export class ServerRecord {
	    UUID: string;
	    Name: string;
	    Path: string;
	    Type: string;
	    Version: string;
	    JavaExecutable: string;
	    JarFile: string;
	    ForgeArgsFile: string;
	    ForgeLaunchType: string;
	    MinRAM: string;
	    MaxRAM: string;
	    JVMArgs: string;
	    UseAikarFlags: boolean;
	    CreatedAt: string;
	    UpdatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new ServerRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.UUID = source["UUID"];
	        this.Name = source["Name"];
	        this.Path = source["Path"];
	        this.Type = source["Type"];
	        this.Version = source["Version"];
	        this.JavaExecutable = source["JavaExecutable"];
	        this.JarFile = source["JarFile"];
	        this.ForgeArgsFile = source["ForgeArgsFile"];
	        this.ForgeLaunchType = source["ForgeLaunchType"];
	        this.MinRAM = source["MinRAM"];
	        this.MaxRAM = source["MaxRAM"];
	        this.JVMArgs = source["JVMArgs"];
	        this.UseAikarFlags = source["UseAikarFlags"];
	        this.CreatedAt = source["CreatedAt"];
	        this.UpdatedAt = source["UpdatedAt"];
	    }
	}

}

export namespace main {
	
	export class DashboardData {
	    totalServers: number;
	    usedGB: number;
	    mostType: string;
	    mostVersion: string;
	
	    static createFrom(source: any = {}) {
	        return new DashboardData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.totalServers = source["totalServers"];
	        this.usedGB = source["usedGB"];
	        this.mostType = source["mostType"];
	        this.mostVersion = source["mostVersion"];
	    }
	}
	export class DeleteResult {
	    success: boolean;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new DeleteResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.message = source["message"];
	    }
	}
	export class JVMConfig {
	    minRAM: string;
	    maxRAM: string;
	    jvmArgs: string;
	    useAikar: boolean;
	    javaExe: string;
	
	    static createFrom(source: any = {}) {
	        return new JVMConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.minRAM = source["minRAM"];
	        this.maxRAM = source["maxRAM"];
	        this.jvmArgs = source["jvmArgs"];
	        this.useAikar = source["useAikar"];
	        this.javaExe = source["javaExe"];
	    }
	}
	export class JavaDownloadOption {
	    name: string;
	    javaVersion: number;
	    mcVersion: string;
	    installed: boolean;
	
	    static createFrom(source: any = {}) {
	        return new JavaDownloadOption(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.javaVersion = source["javaVersion"];
	        this.mcVersion = source["mcVersion"];
	        this.installed = source["installed"];
	    }
	}
	export class JavaRuntimeInfo {
	    name: string;
	    version: number;
	    path: string;
	
	    static createFrom(source: any = {}) {
	        return new JavaRuntimeInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.version = source["version"];
	        this.path = source["path"];
	    }
	}
	export class PropertyEntry {
	    key: string;
	    value: string;
	
	    static createFrom(source: any = {}) {
	        return new PropertyEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.value = source["value"];
	    }
	}
	export class UpdateCheckResult {
	    available: boolean;
	    latest: string;
	    current: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateCheckResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.available = source["available"];
	        this.latest = source["latest"];
	        this.current = source["current"];
	    }
	}

}

