import React from "react";
import {EventsEmit} from "../wailsjs/runtime";

interface ReinitTunnelConfirmProps {
    onExit: () => void;
}

const ReinitTunnelConfirm: React.FC<ReinitTunnelConfirmProps> = ({ onExit }) => {
    const handleNext = async (agree: boolean) => {
        EventsEmit("tunnel_reinit_ask_result", agree);
        onExit();
    };

    return (
        <div className="modal-overlay">
            <div className="modal-container reinit-tunnel-confirm">
                <h2 className="modal-title">Reinitialize Tunnel</h2>
                <div className="modal-message">
                    <p className="title">Tunnel seems stalled, do you want to reinit it?</p>
                    <p className="subtitle">
                        Keep in mind that new payment channels could be opened if the tunnel is not free.
                    </p>
                </div>
                <div className="modal-actions">
                    <button className="button-secondary" onClick={() => handleNext(false)}>No, just wait</button>
                    <button className="button-primary" onClick={() => handleNext(true)}>Yes, reinit</button>
                </div>
            </div>
        </div>
    );
};

export default ReinitTunnelConfirm;