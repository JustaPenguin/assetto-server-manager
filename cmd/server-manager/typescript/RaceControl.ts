import {
    RaceControl as RaceControlData,
    RaceControlDriverMapRaceControlDriver as Driver,
    RaceControlDriverMapRaceControlDriverSessionCarInfo as SessionCarInfo
} from "./models/RaceControl";

import {CarUpdate, CarUpdateVec} from "./models/UDP";
import {randomColor} from "randomcolor/randomColor";
import {msToTime, prettifyName} from "./utils";
import moment = require("moment");
import ClickEvent = JQuery.ClickEvent;

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
    EventRaceControl = 200
;

interface SimpleCollision {
    WorldPos: CarUpdateVec
}

interface WebsocketHandler {
    handleWebsocketMessage(message: WSMessage): void;
}

export class RaceControl {
    private readonly liveMap: LiveMap = new LiveMap(this);
    private readonly liveTimings: LiveTimings = new LiveTimings(this, this.liveMap);
    private readonly $eventTitle: JQuery<HTMLHeadElement>;

    public status: RaceControlData;

    constructor() {
        this.$eventTitle = $("#event-title");
        this.status = new RaceControlData();

        if (!this.$eventTitle.length) {
            return;
        }

        let ws = new WebSocket(((window.location.protocol === "https:") ? "wss://" : "ws://") + window.location.host + "/api/race-control");
        ws.onmessage = this.handleWebsocketMessage.bind(this);
    }

    private handleWebsocketMessage(ev: MessageEvent): void {
        let message = JSON.parse(ev.data) as WSMessage;

        if (!message) {
            return;
        }

        switch (message.EventType) {
            case EventRaceControl:
                this.status = new RaceControlData(message.Message);
                this.$eventTitle.text(RaceControl.getSessionType(this.status.SessionInfo.Type) + " at " + this.status.TrackInfo!.name);
                break;
        }

        this.liveMap.handleWebsocketMessage(message);
        this.liveTimings.handleWebsocketMessage(message);
    }

    private static getSessionType(sessionIndex: number): string {
        switch (sessionIndex) {
            case 0:
                return "Booking";
            case 1:
                return "Practice";
            case 2:
                return "Qualifying";
            case 3:
                return "Race";
            default:
                return "Unknown session";
        }
    }
}

class LiveMap implements WebsocketHandler {
    private mapImageHasLoaded: boolean = false;

    private readonly $map: JQuery<HTMLDivElement>;
    private readonly $trackMapImage: JQuery<HTMLImageElement> | undefined;
    private readonly raceControl: RaceControl;

    constructor(raceControl: RaceControl) {
        this.$map = $("#map");
        this.raceControl = raceControl;
        this.$trackMapImage = this.$map.find("img") as JQuery<HTMLImageElement>;

        $(window).on("resize", this.correctMapDimensions.bind(this));
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

    public handleWebsocketMessage(message: WSMessage): void {
        switch (message.EventType) {
            case EventRaceControl:
                this.trackXOffset = this.raceControl.status.TrackMapData!.offset_x;
                this.trackZOffset = this.raceControl.status.TrackMapData!.offset_y;
                this.trackScale = this.raceControl.status.TrackMapData!.scale_factor;
                this.loadTrackImage();

                for (const connectedGUID in this.raceControl.status.ConnectedDrivers!.Drivers) {
                    const driver = this.raceControl.status.ConnectedDrivers!.Drivers[connectedGUID];

                    if (!this.dots.has(driver.CarInfo.DriverGUID)) {
                        // in the event that a user just loaded the race control page, place the
                        // already loaded dots onto the map
                        let $driverDot = this.buildDriverDot(driver.CarInfo).show();
                        this.dots.set(driver.CarInfo.DriverGUID, $driverDot);
                    }
                }

                break;

            case EventNewConnection:
                const connectedDriver = new SessionCarInfo(message.Message);
                this.dots.set(connectedDriver.DriverGUID, this.buildDriverDot(connectedDriver));

                break;

            case EventClientLoaded:
                let carID = message.Message as number;

                if (!this.raceControl.status!.CarIDToGUID.hasOwnProperty(carID)) {
                    return;
                }

                // find the guid for this car ID:
                this.dots.get(this.raceControl.status!.CarIDToGUID[carID])!.show();

                break;

            case EventConnectionClosed:
                const disconnectedDriver = new SessionCarInfo(message.Message);
                const $dot = this.dots.get(disconnectedDriver.DriverGUID);

                if ($dot) {
                    $dot.hide();
                    this.dots.delete(disconnectedDriver.DriverGUID);
                }

                break;
            case EventCarUpdate:
                const update = new CarUpdate(message.Message);

                if (!this.raceControl.status!.CarIDToGUID.hasOwnProperty(update.CarID)) {
                    return;
                }

                // find the guid for this car ID:
                const driverGUID = this.raceControl.status!.CarIDToGUID[update.CarID];

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
        const out = new CarUpdateVec();

        out.X = ((vec.X + this.trackXOffset + this.trackMargin) / this.trackScale) * this.mapScaleMultiplier;
        out.Z = ((vec.Z + this.trackZOffset + this.trackMargin) / this.trackScale) * this.mapScaleMultiplier;

        return out;
    }

    private buildDriverDot(driverData: SessionCarInfo): JQuery<HTMLElement> {
        if (this.dots.has(driverData.DriverGUID)) {
            return this.dots.get(driverData.DriverGUID)!;
        }

        const $driverName = $("<span class='name'/>").text(getAbbreviation(driverData.DriverName));
        const $info = $("<span class='info'/>").text("0").hide();

        const $dot = $("<div class='dot' style='background: " + randomColor({
            luminosity: 'bright',
            seed: driverData.DriverGUID,
        }) + "'/>").append($driverName, $info).hide().appendTo(this.$map);

        this.dots.set(driverData.DriverGUID, $dot);

        return $dot;
    }

    private getTrackURL(): string {
        if (!this.raceControl.status) {
            return "";
        }

        const sessionInfo = this.raceControl.status.SessionInfo;

        return "/content/tracks/" + sessionInfo.Track + (!!sessionInfo.TrackConfig ? "/" + sessionInfo.TrackConfig : "") + "/map.png";
    }

    trackImage: HTMLImageElement = new Image();

    private loadTrackImage(): void {
        const trackURL = this.getTrackURL();

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

        if (this.trackImage.height / this.trackImage.width > LiveMap.mapRotationRatio) {
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

    public getDotForDriverGUID(guid: string): JQuery<HTMLElement> | undefined {
        return this.dots.get(guid);
    }
}

const DriverGUIDDataKey = "driver-guid";

class LiveTimings implements WebsocketHandler {
    private readonly raceControl: RaceControl;
    private readonly liveMap: LiveMap;

    private readonly $connectedDriversTable: JQuery<HTMLTableElement>;
    private readonly $disconnectedDriversTable: JQuery<HTMLTableElement>;

    constructor(raceControl: RaceControl, liveMap: LiveMap) {
        this.raceControl = raceControl;
        this.liveMap = liveMap;
        this.$connectedDriversTable = $("#live-table");
        this.$disconnectedDriversTable = $("#live-table-disconnected");

        setInterval(this.populateConnectedDrivers.bind(this), 1000);

        $(document).on("click", ".driver-link", this.toggleDriverSpeed.bind(this));
    }

    public handleWebsocketMessage(message: WSMessage): void {
        switch (message.EventType) {
            case EventRaceControl:
                this.populateConnectedDrivers();

                if (!!this.raceControl.status.DisconnectedDrivers) {
                    for (const driverGUID of this.raceControl.status.DisconnectedDrivers!.GUIDsInPositionalOrder) {
                        const driver = this.raceControl.status.DisconnectedDrivers!.Drivers[driverGUID];

                        if (!driver) {
                            continue;
                        }

                        this.addDriverToTable(driver, this.$disconnectedDriversTable);
                    }

                    if (this.raceControl.status.DisconnectedDrivers!.GUIDsInPositionalOrder.length > 0) {
                        this.$disconnectedDriversTable.show();
                    } else {
                        this.$disconnectedDriversTable.hide();
                    }
                }

                break
        }
    }

    private populateConnectedDrivers(): void {
        if (!this.raceControl.status || !this.raceControl.status.ConnectedDrivers) {
            return;
        }

        for (const driverGUID of this.raceControl.status.ConnectedDrivers!.GUIDsInPositionalOrder) {
            const driver = this.raceControl.status.ConnectedDrivers!.Drivers[driverGUID];

            if (!driver) {
                continue;
            }

            this.addDriverToTable(driver, this.$connectedDriversTable);
        }
    }

    // @TODO this should use existing rows and only replace dynamic data, allowing static data to remain unchanged + easy to interact with (e.g. click)
    private addDriverToTable(driver: Driver, $table: JQuery<HTMLTableElement>): void {
        const addingDriverToDisconnectedTable = ($table === this.$disconnectedDriversTable);

        // remove any previous rows
        $("#" + driver.CarInfo.DriverGUID).remove();

        const $tr = $("<tr/>").attr({"id": driver.CarInfo.DriverGUID});

        // car position

        if (!addingDriverToDisconnectedTable) {
            const $tdPos = $("<td class='text-center'/>").text(driver.Position === 255 || driver.Position === 0 ? "" : driver.Position);
            $tr.append($tdPos);
        }

        // driver name
        const $tdName = $("<td/>").text(driver.CarInfo.DriverName);

        if (!addingDriverToDisconnectedTable) {
            // driver dot
            const driverDot = this.liveMap.getDotForDriverGUID(driver.CarInfo.DriverGUID);

            if (driverDot) {
                let dotClass = "dot";

                if (driverDot.find(".info").is(":hidden")) {
                    dotClass += " dot-inactive";
                }

                $tdName.prepend($("<div/>").attr({"class": dotClass}).css("background", randomColor({
                    luminosity: 'bright',
                    seed: driver.CarInfo.DriverGUID,
                })));
            }

            $tdName.attr("class", "driver-link");
            $tdName.data(DriverGUIDDataKey, driver.CarInfo.DriverGUID);
        }

        $tr.append($tdName);

        // car model
        const $tdCar = $("<td/>").text(prettifyName(driver.CarInfo.CarModel, true));
        $tr.append($tdCar);

        if (!addingDriverToDisconnectedTable) {
            let currentLapTimeText = "";

            if (moment(driver.LastLapCompletedTime).isAfter(moment(this.raceControl.status!.SessionStartTime))) {
                // only show current lap time text if the last lap completed time is after session start.
                currentLapTimeText = msToTime(moment().diff(moment(driver.LastLapCompletedTime)), false);
            }

            const $tdCurrentLapTime = $("<td/>").text(currentLapTimeText);
            $tr.append($tdCurrentLapTime);
        }

        // best lap
        const $tdBestLapTime = $("<td/>").text(msToTime(driver.BestLap / 1000000));
        $tr.append($tdBestLapTime);

        if (!addingDriverToDisconnectedTable) {
            // last lap
            const $tdLastLap = $("<td/>").text(msToTime(driver.LastLap / 1000000));
            $tr.append($tdLastLap);

            // gap
            const $tdGap = $("<td/>").text(driver.Split);
            $tr.append($tdGap);
        }

        // lap number
        const $tdLapNum = $("<td/>").text(driver.NumLaps ? driver.NumLaps : "0");
        $tr.append($tdLapNum);

        // @TODO show best AND last
        const $tdTopSpeedBestLap = $("<td/>").text(driver.TopSpeedBestLap ? driver.TopSpeedBestLap.toFixed(2) + "Km/h" : "");
        $tr.append($tdTopSpeedBestLap);

        // events
        const $tdEvents = $("<td/>");

        if (!addingDriverToDisconnectedTable) {
            if (driver.LoadedTime && moment(driver.LoadedTime).add("10s").isSameOrAfter(moment())) {
                // car just loaded
                let $tag = $("<span/>");
                $tag.attr({'class': 'badge badge-success live-badge'});
                $tag.text("Loaded");

                $tdEvents.append($tag);
            }

            if (driver.Collisions) {
                for (const collision of driver.Collisions) {
                    if (collision.Time && moment(collision.Time).add("10s").isSameOrAfter(moment())) {
                        let $tag = $("<span/>");
                        $tag.attr({'class': 'badge badge-danger live-badge'});
                        $tag.text(
                            "Crash " + collision.Type + " at " + collision.Speed.toFixed(2) + "Km/h"
                        );

                        $tdEvents.append($tag);
                    }
                }
            }
        }

        $tr.append($tdEvents);
        $table.append($tr);
    }

    private toggleDriverSpeed(e: ClickEvent): void {
        const $target = $(e.currentTarget);
        const driverGUID = $target.data(DriverGUIDDataKey);
        const $driverDot = this.liveMap.getDotForDriverGUID(driverGUID);

        if (!$driverDot) {
            return;
        }

        $driverDot.find(".info").toggle();
        $target.find(".dot").toggleClass("dot-inactive");
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
