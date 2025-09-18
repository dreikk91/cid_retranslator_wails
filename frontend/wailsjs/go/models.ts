export namespace main {
	
	export class Stats {
	    accepted: number;
	    rejected: number;
	    uptime: string;
	    reconnects: number;
	
	    static createFrom(source: any = {}) {
	        return new Stats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.accepted = source["accepted"];
	        this.rejected = source["rejected"];
	        this.uptime = source["uptime"];
	        this.reconnects = source["reconnects"];
	    }
	}

}

export namespace server {
	
	export class Device {
	    id: number;
	    lastEventTime: string;
	    lastEvent: string;
	
	    static createFrom(source: any = {}) {
	        return new Device(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.lastEventTime = source["lastEventTime"];
	        this.lastEvent = source["lastEvent"];
	    }
	}

}

