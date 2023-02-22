import {useState} from 'react';
import './App.css';
import {StartProxy, StopProxy} from "../wailsjs/go/main/App";
import {EventsOn} from "../wailsjs/runtime";

function App() {
    const [resultText, setResultText] = useState("Not initialized");
    const [resultColor, setResultColor] = useState("#a1a1a1");
    const [buttonDisabled, setButtonDisabled] = useState(false);
    const [isStop, setIsStop] = useState(false);

    EventsOn("statusUpdate", function (typ: string, text: string) {
        let color = "#a1a1a1"
        switch (typ) {
            case "loading":
                color = "greenyellow"
                break
            case "error":
                setButtonDisabled(false);
                color = "red"
                break
            case "ready":
                setIsStop(true);
                setButtonDisabled(false);
                color = "limegreen"
                break
        }

        setResultText(text);
        setResultColor(color);
    })

    async function action() {
        if (buttonDisabled) {
            return
        }

        setButtonDisabled(true);
        if (!isStop) {
            await StartProxy();
        } else {
            await StopProxy();
        }
    }

    return (
        <div id="App">
            <div className="headerTitle">
                <div className="logoIcon">
                    <div style={{height:"70%",width: "70%", margin: "auto"}}>
                        <svg id="Capa_1" xmlns="http://www.w3.org/2000/svg" x="0px" y="0px"
                             viewBox="0 0 214.541 214.541">
                            <g>
                                <g>
                                    <path className="svg-out" d="M205.213,51.897c-0.905-1.6-2.43-2.473-4.28-2.473c-8.031,0-26.566-12.1-44.496-23.796
			C136.241,12.451,117.166,0,106.809,0C95.879,0,73.554,14.215,51.92,27.994c-16.534,10.533-33.638,21.43-39.228,21.43
			c-1.56,0-2.885,0.802-3.625,2.205c-0.483,0.913-0.723,2.065-0.723,3.525c0,20.686,65.439,159.388,98.465,159.388
			c31.183,0,99.388-132.704,99.388-158.078C206.197,54.542,205.882,53.046,205.213,51.897z M106.809,209.166
			c-9.33,0-25.642-17.701-44.746-48.561C34.853,116.649,12.367,63.26,13.644,54.739c7.562-0.805,23.148-10.74,41.228-22.253
			C74.828,19.773,97.447,5.372,106.809,5.372c8.747,0,28.892,13.152,46.697,24.758c19.236,12.551,37.417,24.426,47.162,24.608
			c0.795,1.56,0.204,11.134-11.835,38.394C167.442,141.579,125.891,209.166,106.809,209.166z"/>
                                    <path className="svg-mid" d="M108.974,57.43c-24.351,0-44.156,19.813-44.156,44.163c0,24.361,19.805,44.167,44.156,44.167
			c24.347,0,44.149-19.805,44.149-44.167C153.126,77.242,133.321,57.43,108.974,57.43z M149.326,99.735h-17.579
			c0.064-5.823-0.301-11.621-1.081-17.243c4.341-0.691,8.385-1.56,12.361-2.652C146.831,85.767,149.007,92.649,149.326,99.735z
			 M130.125,78.867c-1.034-6.03-2.308-10.647-3.189-13.453v-0.004l0,0c5.35,2.67,10.125,6.553,13.8,11.23
			C137.061,77.589,133.486,78.337,130.125,78.867z M127.03,83.097c0.741,5.422,1.095,11.016,1.031,16.638h-17.225v-15.6
			C116.526,84.103,121.826,83.767,127.03,83.097L127.03,83.097z M126.457,79.372c-5.136,0.644-10.386,0.977-15.622,0.995V61.241
			c3.976,0.19,7.866,0.938,11.556,2.237v0.004C123.136,65.525,125.036,71.169,126.457,79.372z M149.326,103.45
			c-0.304,6.707-2.294,13.27-5.766,19.014l0,0c-2.308-0.591-7.165-1.761-13.825-2.745c1.045-5.429,1.7-10.901,1.94-16.269
			C131.675,103.45,149.326,103.45,149.326,103.45z M110.835,141.955v-20.056c4.731,0.111,9.581,0.455,14.401,1.031
			c-1.399,5.955-3.439,12.379-5.626,17.654C116.801,141.35,113.845,141.816,110.835,141.955z M126.056,119.219
			c-5.086-0.619-10.214-0.981-15.221-1.099v-14.67h17.136C127.731,108.625,127.09,113.929,126.056,119.219z M124.03,139.106
			c2.051-5.171,3.69-10.45,4.896-15.7c5.905,0.855,10.314,1.84,12.419,2.341l0,0C136.861,131.73,130.88,136.354,124.03,139.106z
			 M87.902,119.594c-4.563,0.701-9.108,1.675-13.496,2.895c-3.482-5.74-5.479-12.322-5.787-19.039h17.368
			C86.231,108.822,86.878,114.255,87.902,119.594z M89.691,103.45h17.433v14.67c-5.318,0.025-10.55,0.354-15.55,0.959
			C90.561,113.854,89.924,108.586,89.691,103.45z M88.704,123.245c1.195,5.254,2.838,10.55,4.878,15.736
			c-6.707-2.766-12.573-7.351-16.978-13.231C80.633,124.68,84.699,123.839,88.704,123.245z M107.124,121.885v20.07
			c-2.935-0.14-5.841-0.594-8.639-1.36c-2.92-6.964-4.978-12.97-6.106-17.855C97.114,122.189,102.067,121.899,107.124,121.885z
			 M107.124,61.241v19.107c-5.275-0.115-10.622-0.512-15.879-1.192c1.41-8.063,3.25-13.553,3.98-15.557
			C99.039,62.222,103.044,61.427,107.124,61.241z M90.668,82.89c5.39,0.705,10.923,1.12,16.456,1.235v15.611H89.609
			C89.541,94.063,89.899,88.394,90.668,82.89z M87.587,78.638c-4.613-0.712-8.203-1.474-10.393-1.983
			c3.618-4.602,8.278-8.428,13.471-11.084C89.824,68.302,88.603,72.787,87.587,78.638z M68.623,99.735
			c0.322-7.043,2.487-13.893,6.259-19.813c2.537,0.648,6.7,1.6,12.132,2.444c-0.795,5.662-1.16,11.502-1.095,17.368
			C85.919,99.735,68.623,99.735,68.623,99.735z"/>
                                </g>
                            </g>
                        </svg>
                    </div>
                </div>
                <p className="logo-text"><b>TON Proxy</b></p>
            </div>
            <div className="content">
                <div>
                    <div className="status">
                        <p><b>127.0.0.1:8080</b></p>
                    </div>
                    <div className="status">
                        <p><b id="status" style={{color: resultColor}}>{resultText}</b></p>
                    </div>
                    <div className="button-block">
                        <div id="start" className={buttonDisabled ? "startBtnContainer disabled-button": "startBtnContainer"} onClick={action}>
                            {isStop ? "Stop Gateway" : "Start Gateway"}
                        </div>
                    </div>
                </div>
            </div>
            <div className="footerTitle">
                <div style={{display: "flex", justifyContent: "center", justifyItems: "center"}}>
                    <p className="foot-text"><b>Developed with ❤️ by <a className="utils-link"
                                                                        href="#">Tonutils</a> team</b></p>
                    <p className="ver-text"><b>v0.3.0</b></p>
                </div>
            </div>
        </div>
    )
}

export default App