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
	
	export class Event {
	    time: string;
	    data: string;
	
	    static createFrom(source: any = {}) {
	        return new Event(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.time = source["time"];
	        this.data = source["data"];
	    }
	}
	export class Device {
	    id: number;
	    lastEventTime: string;
	    lastEvent: string;
	    events: Event[];
	
	    static createFrom(source: any = {}) {
	        return new Device(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.lastEventTime = source["lastEventTime"];
	        this.lastEvent = source["lastEvent"];
	        this.events = this.convertValues(source["events"], Event);
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
	
	export class GlobalEvent {
	    time: string;
	    deviceID: number;
	    data: string;
	
	    static createFrom(source: any = {}) {
	        return new GlobalEvent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.time = source["time"];
	        this.deviceID = source["deviceID"];
	        this.data = source["data"];
	    }
	}

}

