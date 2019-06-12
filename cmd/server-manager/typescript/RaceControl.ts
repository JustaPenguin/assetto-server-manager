import {
    RaceControl as RaceControlData,
    RaceControlDriverMapRaceControlDriverSessionCarInfo as SessionCarInfo
} from "./models/RaceControl";

import {CarUpdate, CarUpdateVec} from "./models/UDP";
import {randomColor} from "randomcolor/randomColor";

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

interface SimpleCollision {
    WorldPos: CarUpdateVec
}

export class RaceControl {
    status?: RaceControlData;

    private mapImageHasLoaded: boolean = false;

    private readonly $map: JQuery<HTMLDivElement>;
    private readonly $trackMapImage: JQuery<HTMLImageElement> | undefined;

    constructor() {
        this.$map = $("#map");

        if (!this.$map.length) {
            return; // no live map
        }

        this.$trackMapImage = this.$map.find("img") as JQuery<HTMLImageElement>;

        $(window).on("resize", this.correctMapDimensions.bind(this));

        let ws = new WebSocket(((window.location.protocol === "https:") ? "wss://" : "ws://") + window.location.host + "/api/race-control");
        ws.onmessage = this.handleWebsocketMessage.bind(this);
    }

    // positional coordinate modifiers.
    private mapScaleMultiplier: number = 1;
    private trackScale: number = 1;
    private trackMargin: number = 0;
    private trackXOffset: number = 0;
    private trackZOffset: number = 0;

    // live map track dots
    private dots: Map<string, JQuery<HTMLElement>> = new Map<string, JQuery<HTMLElement>>();
    private maxRPMs: Map<string, number> = new Map<string, number>();

    private handleWebsocketMessage(ev: MessageEvent): void {
        let message = JSON.parse(ev.data) as WSMessage;

        if (!message) {
            return;
        }

        switch (message.EventType) {
            case EventRaceControl:
                this.status = message.Message as RaceControlData;
                this.trackXOffset = this.status.TrackMapData!.offset_x;
                this.trackZOffset = this.status.TrackMapData!.offset_y;
                this.trackScale = this.status.TrackMapData!.scale_factor;
                this.loadTrackImage();

                for (const connectedGUID in this.status.ConnectedDrivers!.Drivers) {
                    const driver = this.status.ConnectedDrivers!.Drivers[connectedGUID];

                    if (!this.dots.has(driver.CarInfo.DriverGUID)) {
                        // in the event that a user just loaded the race control page, place the
                        // already loaded dots onto the map
                        let $driverDot = this.buildDriverDot(driver.CarInfo).show();
                        this.dots.set(driver.CarInfo.DriverGUID, $driverDot);
                    }
                }
                break;


            case EventNewConnection:
                const connectedDriver = message.Message as SessionCarInfo;
                this.dots.set(connectedDriver.DriverGUID, this.buildDriverDot(connectedDriver));

                // @TODO hide this by default.

                break;

            case EventClientLoaded:
                let carID = message.Message as number;

                if (!this.status!.CarIDToGUID.hasOwnProperty(carID)) {
                    return;
                }

                // find the guid for this car ID:
                this.dots.get(this.status!.CarIDToGUID[carID])!.show();

                break;

            case EventConnectionClosed:
                const disconnectedDriver = message.Message as SessionCarInfo;
                const $dot = this.dots.get(disconnectedDriver.DriverGUID);

                if ($dot) {
                    $dot.hide();
                    this.dots.delete(disconnectedDriver.DriverGUID);
                }

                break;
            case EventCarUpdate:
                const update = message.Message as CarUpdate;

                if (!this.status!.CarIDToGUID.hasOwnProperty(update.CarID)) {
                    return;
                }

                // find the guid for this car ID:
                const driverGUID = this.status!.CarIDToGUID[update.CarID];

                let $myDot = this.dots.get(driverGUID);
                let dotPos = this.translateToTrackCoordinate(update.Pos);

                $myDot!.css({
                    "left": dotPos.X,
                    "top": dotPos.Z,
                });

                let speed = Math.floor(Math.sqrt((Math.pow(update.Velocity.X, 2) + Math.pow(update.Velocity.Z, 2))) * 3.6);

                let maxRPM = this.maxRPMs.get(driverGUID);

                if (!maxRPM) {
                    maxRPM = 0;
                }

                if (update.EngineRPM > maxRPM) {
                    maxRPM = update.EngineRPM;
                    this.maxRPMs.set(driverGUID, update.EngineRPM);
                }

                let $rpmGaugeOuter = $("<div class='rpm-outer'></div>");
                let $rpmGaugeInner = $("<div class='rpm-inner'></div>");

                $rpmGaugeInner.css({
                    'width': ((update.EngineRPM / maxRPM) * 100).toFixed(0) + "%",
                    'background': randomColor({
                        luminosity: 'bright',
                        seed: driverGUID,
                    }),
                });

                $rpmGaugeOuter.append($rpmGaugeInner);
                $myDot!.find(".info").text(speed + "Km/h " + (update.Gear - 1));
                $myDot!.find(".info").append($rpmGaugeOuter);
                break;

            case EventVersion:
                // completely new server instance, refresh the page.
                // @TODO this might not be necessary.
                location.reload();
                break;

            case EventNewSession:
                this.loadTrackImage();

                break;

            case EventCollisionWithCar:
            case EventCollisionWithEnv:
                let collisionData = message.Message as SimpleCollision;

                let collisionMapPoint = this.translateToTrackCoordinate(collisionData.WorldPos);

                let $collision = $("<div class='collision' />").css({
                    'left': collisionMapPoint.X,
                    'top': collisionMapPoint.Z,
                });

                $collision.appendTo(this.$map);

                break;
        }
    }

    private translateToTrackCoordinate(vec: CarUpdateVec): CarUpdateVec {
        let out = new CarUpdateVec();

        out.X = ((vec.X + this.trackXOffset + this.trackMargin) / this.trackScale) * this.mapScaleMultiplier;
        out.Z = ((vec.Z + this.trackZOffset + this.trackMargin) / this.trackScale) * this.mapScaleMultiplier;

        return out;
    }

    private buildDriverDot(driverData: SessionCarInfo): JQuery<HTMLElement> {
        if (this.dots.has(driverData.DriverGUID)) {
            return this.dots.get(driverData.DriverGUID)!;
        }

        let $driverName = $("<span class='name'/>").text(getAbbreviation(driverData.DriverName));
        let $info = $("<span class='info'/>").text("0");

        let $dot = $("<div class='dot' style='background: " + randomColor({
            luminosity: 'bright',
            seed: driverData.DriverGUID,
        }) + "'/>").append($driverName, $info).hide().appendTo(this.$map);

        this.dots.set(driverData.DriverGUID, $dot);

        return $dot;
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
        let trackURL = this.getTrackURL();

        this.trackImage.onload = () => {
            this.$trackMapImage!.attr({
                "src": trackURL,
            });

            this.mapImageHasLoaded = true;
            this.correctMapDimensions();
        };

        this.trackImage.src = trackURL
    }

    private static mapRotationRatio: number = 1.07;

    private correctMapDimensions(): void {
        if (!this.trackImage || !this.$trackMapImage || !this.mapImageHasLoaded) {
            return;
        }

        if (this.trackImage.height / this.trackImage.width > RaceControl.mapRotationRatio) {
            // rotate the map
            this.$map.addClass("rotated");

            this.$trackMapImage.css({
                'max-height': this.$trackMapImage.closest(".map-container").width()!,
                'max-width': 'auto'
            });

            this.mapScaleMultiplier = this.$trackMapImage.width()! / this.trackImage.width;

            this.$map.closest(".map-container").css({
                'max-height': (this.trackImage.width * this.mapScaleMultiplier) + 20,
            });

            this.$map.css({
                'max-width': (this.trackImage.width * this.mapScaleMultiplier) + 20,
            });
        } else {
            // un-rotate the map
            this.$map.removeClass("rotated").css({
                'max-height': 'inherit',
                'max-width': '100%',
            });

            this.$map.closest(".map-container").css({
                'max-height': 'auto',
            });

            this.$trackMapImage.css({
                'max-height': 'inherit',
                'max-width': '100%'
            });

            this.mapScaleMultiplier = this.$trackMapImage.width()! / this.trackImage.width;
        }
    }
}

function getAbbreviation(name: string): string {
    let parts = name.split(" ");

    if (parts.length < 1) {
        return name
    }

    let lastName = parts[parts.length - 1];

    return lastName.slice(0, 3).toUpperCase();
}
