export namespace store {
	
	export class User {
	    id: number;
	    name: string;
	    password: string;
	
	    static createFrom(source: any = {}) {
	        return new User(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.password = source["password"];
	    }
	}

}

