import React from "react";
import {EventsEmit} from "../wailsjs/runtime";

interface ResetTunnelConfirmProps {
    onCancel: () => void;
    onReset: () => void;
}

const ResetTunnelConfirm: React.FC<ResetTunnelConfirmProps> = ({ onReset, onCancel }) => {
    return (
        <div className="modal-overlay">
            <div className="modal-container reinit-tunnel-confirm">
                <h2 className="modal-title">Tunnel</h2>
                <div className="modal-message">
                    <p className="title">Do you want to reset config?</p>
                    <p className="subtitle">
                        After reset you may select config file again.
                    </p>
                </div>
                <div className="modal-actions">
                    <button className="button-secondary" onClick={() => onCancel()}>Cancel</button>
                    <button className="button-primary" onClick={() => onReset()}>Reset tunnel config</button>
                </div>
            </div>
        </div>
    );
};

export default ResetTunnelConfirm;