import {RaceControl as RaceControlData} from "./models/RaceControl";
import {CarUpdate} from "./models/UDP";

interface WSMessage {
    Message: any;
    EventType: number;
}

const EventCollisionWithCar = 10,
    EventCollisionWithEnv = 11,
    EventNewSession = 50,
    EventNewConnection = 51,
    EventConnectionClosed = 52,
    EventCarUpdate = 53,
    EventCarInfo = 54,
    EventEndSession = 55,
    EventVersion = 56,
    EventChat = 57,
    EventClientLoaded = 58,
    EventSessionInfo = 59,
    EventError = 60,
    EventLapCompleted = 73,
    EventClientEvent = 130,
    EventTrackMapInfo = 222,
    EventRaceControl = 200
;

export class RaceControl {

    status?: RaceControlData;

    private mapScaleMultiplier: number = 1;
    private scale: number = 1;
    private margin: number = 0;
    private mapImageHasLoaded: boolean = false;

    constructor() {
        let ws = new WebSocket(((window.location.protocol === "https:") ? "wss://" : "ws://") + window.location.host + "/api/race-control");

        ws.onmessage = this.handleWebsocketMessage;
    }

    private handleWebsocketMessage(ev: MessageEvent): void {
        console.log(ev);

        let message = JSON.parse(ev.data) as WSMessage;

        if (!message) {
            return;
        }

        switch (message.EventType) {
            case EventRaceControl:
                this.status = message.Message as RaceControlData;

                break;

            case EventNewConnection:

                break;

            case EventConnectionClosed:

                break;
            case EventCarUpdate:
                let update = message.Message as CarUpdate;

                let speed = Math.floor(Math.sqrt((Math.pow(update.Velocity.X, 2) + Math.pow(update.Velocity.Z, 2))) * 3.6);

                break;

            case EventVersion:
                // completely new server instance, refresh the page.
                // @TODO this might not be necessary.
                location.reload();
                break;

            case EventNewSession:

                break;

            case EventCollisionWithCar:
            case EventCollisionWithEnv:

                break;
        }
    }

    private getTrackURL(): string {
        if (!this.status) {
            return "";
        }

        let sessionInfo = this.status.SessionInfo;

        return "/content/tracks/" + sessionInfo.Track + (!!sessionInfo.TrackConfig ? "/" + sessionInfo.TrackConfig : "") + "/map.png";
    }

    trackImage: HTMLImageElement = new Image();

    private loadTrackImage(): void {
        this.trackImage.onload = () => {

        }
    }

}