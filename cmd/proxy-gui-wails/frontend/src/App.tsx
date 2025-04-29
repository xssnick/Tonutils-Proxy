import {useEffect, useState} from 'react';
import './App.css';
import {AddTunnel, GetProxyAddr, GetTunnelEnabled, StartProxy, StopProxy} from "../wailsjs/go/main/App";
import {BrowserOpenURL, EventsEmit, EventsOn} from "../wailsjs/runtime";
import Logo from "./assets/logo.svg";
import TunnelNodesModal from "./TunnelNodesModal";
import ReinitTunnelConfirm from "./ReinitTunnelConfirm";
import TunnelConfigurationModal from "./TunnelConfigurationModal";
import {main} from "../wailsjs/go/models";
import SectionInfo = main.SectionInfo; // Импортируем модалку

interface TunnelData {
    sections: SectionInfo[];
    priceIn: string;
    priceOut: string;
}

interface TunnelPoolData {
    path: string;
    max: number;
}

function App() {
    const [resultText, setResultText] = useState("Disconnected");
    const [proxyState, setProxyState] = useState("stopped");
    const [buttonDisabled, setButtonDisabled] = useState(false);
    const [isStop, setIsStop] = useState(false);
    const [hasTunnel, setHasTunnel] = useState(false);
    const [isTunnelModalOpen, setIsTunnelModalOpen] = useState(false);
    const [isTunnelConfigModalOpen, setIsTunnelConfigModalOpen] = useState(false);
    const [isTunnelReinitModalOpen, setIsTunnelReinitModalOpen] = useState(false);

    const [paidTunnel, setPaidTunnel] = useState("");
    const [proxyAddress, setProxyAddress] = useState("127.0.0.1:8080");
    const [tunnelAddress, setTunnelAddress] = useState("");
    const [tunnelData, setTunnelData] = useState<TunnelData | null>(null);
    const [tunnelPoolData, setTunnelPoolData] = useState<TunnelPoolData | null>(null);

    EventsOn("tunnel_updated", (addr: string)=> {
        setTunnelAddress(addr);
    })
    EventsOn("tunnel_check", (sections: SectionInfo[], priceIn: string, priceOut: string)=> {
        setTunnelData({
            sections,
            priceIn,
            priceOut,
        });
        
        setIsTunnelModalOpen(true);
    })
    EventsOn("tunnel_reinit_ask", ()=> {
        setIsTunnelReinitModalOpen(true);
    })
    EventsOn("tunnel_paid", (amt: string)=> {
        console.log("tunnel paid updated", amt);
        setPaidTunnel(amt);
    })
    EventsOn("tunnel_pool_added", function (path: string, max: number) {
        if (path == "" || max == 0) return;

        setTunnelPoolData({
            path,
            max,
        });
        setIsTunnelConfigModalOpen(true);
    })
    EventsOn("statusUpdate", function (typ: string, text: string) {
        switch (typ) {
            case "loading":
                setProxyState("loading");
                if (text === "stopping") {
                    text = "Disconnecting..."
                    break;
                }
                //text = text === "stopping" ? "DISCONNECTING..." : "CONNECTING...";
                break;
            case "error":
                setProxyState("error");
                setButtonDisabled(false);
                break;
            case "ready":
                setIsStop(true);
                setButtonDisabled(false);
                setProxyState("ready");
                text = "Connected";
                break;
            case "stopped":
                setPaidTunnel("");
                setTunnelAddress("");
                setIsStop(false);
                setButtonDisabled(false);
                setProxyState("stopped");
                text = "Disconnected";
                break;
        }
        setResultText(text);
    });
    EventsOn("config_saved", function () {
        GetTunnelEnabled().then((enabled: boolean) => setHasTunnel(enabled));
    })

    async function action() {
        if (buttonDisabled) {
            return;
        }
        setButtonDisabled(true);
        if (!isStop) {
            await StartProxy();
        } else {
            await StopProxy();
        }
    }

    function toButtonState(state: string) {
        if (state === 'ready') return 'state-connected';
        if (state === 'loading') return 'state-connecting';
        return 'state-not-connected';
    }

    useEffect(() => {
        GetTunnelEnabled().then((enabled: boolean) => setHasTunnel(enabled));
        GetProxyAddr().then((addr: string) => setProxyAddress(addr));
    }, []);

    return (
        <div className={"app " + (proxyState === 'ready' ? "map-active" : "map-inactive")}>
            <div className="logo-container">
                <img src={Logo} className="logo" alt=""/>
            </div>
            <div className="content-container">
                <div className="ip-container">
                    <label>Proxy IP</label>
                    <label className="big-text">{proxyAddress}</label>
                    {hasTunnel && tunnelAddress && (
                        <label className="small-text-tunnel">-&gt; {tunnelAddress}</label>
                    )}
                </div>
                <div className="button-container">
                    <button className={"button " + toButtonState(proxyState)} onClick={action}/>
                </div>
                <button
                    className={`apply-config-button ${isStop ? 'already-started' : `${hasTunnel ? "applied" : ""}`}`}
                    disabled={isStop}
                    onClick={AddTunnel}
                >
                    {isStop ? "Stop to edit tunnel" : `${hasTunnel ? "Tunnel Config Applied" : "Apply Tunnel Config"}`}
                </button>
                {paidTunnel && hasTunnel && (
                    <div className="small-text-paid">
                        Paid: {paidTunnel} TON
                    </div>
                )}
                <label className={"big-text "+(paidTunnel && hasTunnel ? "status-upper" : "status")}>{resultText}</label>
            </div>
            <div className="author-container">
                <label className="small-text">v1.7.0</label>
            </div>

            {isTunnelModalOpen && (
                <TunnelNodesModal
                    sections={tunnelData!.sections}
                    pricePerMBIn={tunnelData!.priceIn}
                    pricePerMBOut={tunnelData!.priceOut}
                    onCancel={() => {
                        EventsEmit("tunnel_check_result");
                        setIsTunnelModalOpen(false);
                    }}
                    onReroute={() => {
                        EventsEmit("tunnel_check_result", false);
                        setIsTunnelModalOpen(false);
                    }}
                    onAccept={() => {
                        EventsEmit("tunnel_check_result", true);
                        setIsTunnelModalOpen(false);
                    }}
                />
            )}

            {isTunnelConfigModalOpen && (
                <TunnelConfigurationModal
                    max={tunnelPoolData!.max}
                    maxFree={tunnelPoolData!.max}
                    poolPath={tunnelPoolData!.path}
                    onClose={() => setIsTunnelConfigModalOpen(false)}
                />
            )}

            {isTunnelReinitModalOpen && (
                <ReinitTunnelConfirm
                    onExit={() => setIsTunnelReinitModalOpen(false)}
                />
            )}
        </div>
    );
}

export default App;