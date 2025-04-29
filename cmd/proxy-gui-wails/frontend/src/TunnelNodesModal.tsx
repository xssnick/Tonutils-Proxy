import React from "react";
import SectionInfo = main.SectionInfo;
import {main} from "../wailsjs/go/models";

interface TunnelNodesModalProps {
    sections: SectionInfo[];
    pricePerMBIn: string;
    pricePerMBOut: string;
    onCancel: () => void;
    onReroute: () => void;
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
    return (
        <div className="modal-overlay">
            <div className="modal-container tunnel-nodes-container">
                <h2 className="modal-title">Tunnel Route</h2>

                <div className="nodes-route-horizontal">
                    {sections.map((node, idx) => (
                        <React.Fragment key={idx}>
                            <span className={`node${node.Outer ? " outer" : ""}`}>
                                {node.Name}
                            </span>
                            {idx !== sections.length - 1 && <span className="arrow">→</span>} {/* Стрелка между узлами */}
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

                <div className="modal-actions">
                    <button className="button-secondary small-button" onClick={onCancel}>
                        Cancel
                    </button>
                    <button className="button-secondary small-button" onClick={onReroute}>
                        Reroute
                    </button>
                    <button className="button-primary small-button" onClick={onAccept}>
                        Accept
                    </button>
                </div>
            </div>
        </div>
    );
};

export default TunnelNodesModal;