import React from "react";
import {QRCodeSVG} from "qrcode.react";

interface QRData {
    address: string;
}

const QR: React.FC<QRData> = ({address}) => {
    return (
        <div style={{ textAlign: "center",marginTop: "10px"}}>
            <QRCodeSVG fgColor={"white"} bgColor={"transparent"} value={address} size={38} />
        </div>
    );
};

export default QR;