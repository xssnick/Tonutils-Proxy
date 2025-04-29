export namespace config {
	
	export class BalanceControlConfig {
	    DepositWhenAmountLessThan: string;
	    DepositUpToAmount: string;
	    WithdrawWhenAmountReached: string;
	
	    static createFrom(source: any = {}) {
	        return new BalanceControlConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.DepositWhenAmountLessThan = source["DepositWhenAmountLessThan"];
	        this.DepositUpToAmount = source["DepositUpToAmount"];
	        this.WithdrawWhenAmountReached = source["WithdrawWhenAmountReached"];
	    }
	}
	export class VirtualConfig {
	    ProxyMaxCapacity: string;
	    ProxyMinFee: string;
	    ProxyFeePercent: number;
	    AllowTunneling: boolean;
	
	    static createFrom(source: any = {}) {
	        return new VirtualConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ProxyMaxCapacity = source["ProxyMaxCapacity"];
	        this.ProxyMinFee = source["ProxyMinFee"];
	        this.ProxyFeePercent = source["ProxyFeePercent"];
	        this.AllowTunneling = source["AllowTunneling"];
	    }
	}
	export class CoinConfig {
	    Enabled: boolean;
	    VirtualTunnelConfig: VirtualConfig;
	    MisbehaviorFine: string;
	    ExcessFeeTon: string;
	    Symbol: string;
	    Decimals: number;
	    BalanceControl?: BalanceControlConfig;
	
	    static createFrom(source: any = {}) {
	        return new CoinConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Enabled = source["Enabled"];
	        this.VirtualTunnelConfig = this.convertValues(source["VirtualTunnelConfig"], VirtualConfig);
	        this.MisbehaviorFine = source["MisbehaviorFine"];
	        this.ExcessFeeTon = source["ExcessFeeTon"];
	        this.Symbol = source["Symbol"];
	        this.Decimals = source["Decimals"];
	        this.BalanceControl = this.convertValues(source["BalanceControl"], BalanceControlConfig);
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
	export class CoinTypes {
	    Ton: CoinConfig;
	    Jettons: Record<string, CoinConfig>;
	    ExtraCurrencies: Record<number, CoinConfig>;
	
	    static createFrom(source: any = {}) {
	        return new CoinTypes(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Ton = this.convertValues(source["Ton"], CoinConfig);
	        this.Jettons = this.convertValues(source["Jettons"], CoinConfig, true);
	        this.ExtraCurrencies = this.convertValues(source["ExtraCurrencies"], CoinConfig, true);
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
	export class ChannelsConfig {
	    SupportedCoins: CoinTypes;
	    BufferTimeToCommit: number;
	    QuarantineDurationSec: number;
	    ConditionalCloseDurationSec: number;
	    MinSafeVirtualChannelTimeoutSec: number;
	
	    static createFrom(source: any = {}) {
	        return new ChannelsConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.SupportedCoins = this.convertValues(source["SupportedCoins"], CoinTypes);
	        this.BufferTimeToCommit = source["BufferTimeToCommit"];
	        this.QuarantineDurationSec = source["QuarantineDurationSec"];
	        this.ConditionalCloseDurationSec = source["ConditionalCloseDurationSec"];
	        this.MinSafeVirtualChannelTimeoutSec = source["MinSafeVirtualChannelTimeoutSec"];
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
	export class PaymentsClientConfig {
	    ADNLServerKey: number[];
	    PaymentsNodeKey: number[];
	    WalletPrivateKey: number[];
	    DBPath: string;
	    SecureProofPolicy: boolean;
	    ChannelsConfig: ChannelsConfig;
	
	    static createFrom(source: any = {}) {
	        return new PaymentsClientConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ADNLServerKey = source["ADNLServerKey"];
	        this.PaymentsNodeKey = source["PaymentsNodeKey"];
	        this.WalletPrivateKey = source["WalletPrivateKey"];
	        this.DBPath = source["DBPath"];
	        this.SecureProofPolicy = source["SecureProofPolicy"];
	        this.ChannelsConfig = this.convertValues(source["ChannelsConfig"], ChannelsConfig);
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
	export class ClientConfig {
	    TunnelServerKey: number[];
	    TunnelThreads: number;
	    TunnelSectionsNum: number;
	    NodesPoolConfigPath: string;
	    PaymentsEnabled: boolean;
	    Payments: PaymentsClientConfig;
	
	    static createFrom(source: any = {}) {
	        return new ClientConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.TunnelServerKey = source["TunnelServerKey"];
	        this.TunnelThreads = source["TunnelThreads"];
	        this.TunnelSectionsNum = source["TunnelSectionsNum"];
	        this.NodesPoolConfigPath = source["NodesPoolConfigPath"];
	        this.PaymentsEnabled = source["PaymentsEnabled"];
	        this.Payments = this.convertValues(source["Payments"], PaymentsClientConfig);
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
	
	export class Config {
	    Version: number;
	    ProxyListenAddr: string;
	    ADNLKey: number[];
	    NetworkConfigPath: string;
	    CustomTunnelNetworkConfigPath: string;
	    TunnelConfig?: config.ClientConfig;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Version = source["Version"];
	        this.ProxyListenAddr = source["ProxyListenAddr"];
	        this.ADNLKey = source["ADNLKey"];
	        this.NetworkConfigPath = source["NetworkConfigPath"];
	        this.CustomTunnelNetworkConfigPath = source["CustomTunnelNetworkConfigPath"];
	        this.TunnelConfig = this.convertValues(source["TunnelConfig"], config.ClientConfig);
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
	export class SectionInfo {
	    Name: string;
	    Outer: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SectionInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Name = source["Name"];
	        this.Outer = source["Outer"];
	    }
	}

}

