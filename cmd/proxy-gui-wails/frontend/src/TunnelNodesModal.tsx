import React, {useEffect, useState} from "react";
import SectionInfo = main.SectionInfo;
import {main} from "../wailsjs/go/models";
import {GetConfig, GetMaxTunnelNodes, GetPaymentNetworkWalletAddr, SaveTunnelConfig} from "../wailsjs/go/main/App";
import Config = main.Config;

interface TunnelNodesModalProps {
    sections: SectionInfo[];
    pricePerMBIn: string;
    pricePerMBOut: string;
    onCancel: () => void;
    onReroute: (num: number) => void;
    onAccept: () => void;
}

export const TunnelNodesModal: React.FC<TunnelNodesModalProps> = ({
                                                                      sections,
                                                                      pricePerMBIn,
                                                                      pricePerMBOut,
                                                                      onCancel,
                                                                      onReroute,
                                                                      onAccept,
                                                                  }) => {

    const [nodes, setNodes] = useState<number>(0);
    const [addr, setAddr] = useState<string>("Loading...");
    const [maxNodes, setMaxNodes] = useState<number>(0);

    const handleIncrementNodes = () => {
        let num = Math.min(nodes + 1, maxNodes);
        if (num != nodes)  {
            onReroute(num);
        }
        setNodes(num);
    };

    const handleDecrementNodes = () => {
        let num = Math.max(nodes - 1, 1);
        if (num != nodes)  {
            onReroute(num);
        }
        setNodes(num);
    };

    React.useEffect(() => {
        GetMaxTunnelNodes().then((num: number) => {
            setMaxNodes(num);
            GetConfig().then((cfg: Config) => {
                setNodes(Math.min(num, cfg.TunnelConfig!.TunnelSectionsNum));
            });
        });
        GetPaymentNetworkWalletAddr().then((addr: string) => {
            setAddr(addr);
        });
    }, []);

    return (
        <div className="modal-overlay">
            <div className="modal-container tunnel-nodes-container">
                <h2 className="modal-title">Tunnel Route</h2>

                <div className="field-group">
                    <div className="nodes-control">
                        <button
                            onClick={handleDecrementNodes}
                            className="nodes-button"
                            disabled={nodes <= 1}
                        >
                            –
                        </button>
                        <span className="nodes-value">{nodes}</span>
                        <button
                            onClick={handleIncrementNodes}
                            className="nodes-button"
                            disabled={nodes >= maxNodes}
                        >
                            +
                        </button>
                    </div>
                </div>

                <div className="nodes-route-horizontal">
                    {sections.map((node, idx) => (
                        <React.Fragment key={idx}>
                            <span className={`node${node.Outer ? " outer" : ""}`}>
                                {node.Name}
                            </span>
                            {idx !== sections.length - 1 && <span className="arrow">→</span>}
                        </React.Fragment>
                    ))}
                </div>

                <div className="prices">
                    <div className="price-item">
                        <span className="price-label">Price per MB (In):</span>
                        <span className="price-value">{pricePerMBIn} TON</span>
                    </div>
                    <div className="price-item">
                        <span className="price-label">Price per MB (Out):</span>
                        <span className="price-value">{pricePerMBOut} TON</span>
                    </div>
                </div>

                <div className="field-group">
                    {(pricePerMBIn !== "0" || pricePerMBOut !== "0") && (
                        <div className="ton-address-group">
                            <input
                                id="tonAddress"
                                className="ton-address-input"
                                type="text"
                                readOnly
                                value={addr}
                                onClick={(e) => e.currentTarget.select()}
                            />
                            <p className="important-note">
                                Deposit at least 5.5 TON for tunnel payments.
                            </p>
                        </div>
                    )}
                </div>

                <div className="modal-actions">
                    <button className="button-secondary small-button" onClick={onCancel}>
                        Cancel
                    </button>
                    <button className="button-secondary small-button" onClick={() => {onReroute(nodes)}} disabled={nodes <= 0}>
                        Reroute
                    </button>
                    <button className="button-primary small-button" onClick={onAccept} disabled={nodes <= 0}>
                        Accept
                    </button>
                </div>
            </div>
        </div>
    );
};

export default TunnelNodesModal;