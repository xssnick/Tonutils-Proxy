import React, { useState, useEffect } from "react";
import {main} from "../wailsjs/go/models";
import Config = main.Config;
import {GetConfig, GetPaymentNetworkWalletAddr, SaveTunnelConfig} from "../wailsjs/go/main/App";

interface TunnelConfigurationModalProps {
    onClose: () => void;
    max: number;
    maxFree: number;
    poolPath: string;
}

const TunnelConfigurationModal: React.FC<TunnelConfigurationModalProps> = ({
                                                                               onClose,
                                                                               max,
                                                                               maxFree,
                                                                               poolPath,
                                                                           }) => {
    const [nodes, setNodes] = useState<number>(1);
    const [addr, setAddr] = useState<string>("Loading...");
    const [enablePayments, setEnablePayments] = useState<boolean>(true);

    const handleIncrementNodes = () => setNodes((prev) => Math.min(prev + 1, max));
    const handleDecrementNodes = () => setNodes((prev) => Math.max(prev - 1, 1));

    const isSaveDisabled = nodes > max;

    const handleSave = () => {
        if (!isSaveDisabled) {
            SaveTunnelConfig(nodes, enablePayments, poolPath).then(() => {
                onClose();
            });
        }
    };

   useEffect(() => {
        GetConfig().then((cfg: Config) => {
            setNodes(cfg.TunnelConfig?.TunnelSectionsNum || 1);
            setEnablePayments(cfg.TunnelConfig?.PaymentsEnabled || false);
        });
        GetPaymentNetworkWalletAddr().then((addr: string) => setAddr(addr));
    }, []);

    return (
        <div className="modal-overlay">
            <div className="modal-container">
                <h2 className="modal-title">Tunnel Configuration</h2>

                <div className="modal-content">
                    <div className="field-group">
                        <label className="field-label">Number of Nodes</label>
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
                                disabled={nodes >= max}
                            >
                                +
                            </button>
                        </div>
                    </div>

                    <div className="field-group">
                        <label className="checkbox-container">
                            <input
                                type="checkbox"
                                checked={enablePayments}
                                onChange={(e) => setEnablePayments(e.target.checked)}
                            />
                            Enable Payments
                        </label>

                        {enablePayments && (
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
                                    Deposit at least 5 TON for tunnel payments.
                                </p>
                            </div>
                        )}
                    </div>
                </div>

                <div className="modal-actions">
                    <button className="button-secondary" onClick={onClose}>
                        Cancel
                    </button>
                    <button
                        className="button-primary"
                        onClick={handleSave}
                        disabled={isSaveDisabled}
                    >
                        Save
                    </button>
                </div>
            </div>
        </div>
    );
};

export default TunnelConfigurationModal;