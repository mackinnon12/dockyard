export namespace database {
	
	export class Container {
	    id: string;
	    name: string;
	    image: string;
	    state: string;
	    status: string;
	    publicAddress: string;
	    publicPort: string;
	
	    static createFrom(source: any = {}) {
	        return new Container(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.image = source["image"];
	        this.state = source["state"];
	        this.status = source["status"];
	        this.publicAddress = source["publicAddress"];
	        this.publicPort = source["publicPort"];
	    }
	}
	export class MySQLConfig {
	    name: string;
	    version: string;
	    port: number;
	    rootPassword: string;
	    database: string;
	
	    static createFrom(source: any = {}) {
	        return new MySQLConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.version = source["version"];
	        this.port = source["port"];
	        this.rootPassword = source["rootPassword"];
	        this.database = source["database"];
	    }
	}

}

