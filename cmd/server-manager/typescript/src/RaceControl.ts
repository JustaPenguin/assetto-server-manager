import {
    RaceControl as RaceControlData,
    RaceControlDriverMapRaceControlDriver as Driver,
    RaceControlDriverMapRaceControlDriverSessionCarInfo as SessionCarInfo
} from "./models/RaceControl";

import {CarUpdate, CarUpdateVec} from "./models/UDP";
import {randomColor} from "randomcolor/randomColor";
import {msToTime, prettifyName} from "./utils";
import moment from "moment";
import ClickEvent = JQuery.ClickEvent;
import ChangeEvent = JQuery.ChangeEvent;
import ReconnectingWebSocket from "reconnecting-websocket";

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
    onTrackChange(track: string, trackLayout: string): void;
}

export class RaceControl {
    private readonly liveMap: LiveMap = new LiveMap(this);
    private readonly liveTimings: LiveTimings = new LiveTimings(this, this.liveMap);
    private readonly $eventTitle: JQuery<HTMLHeadElement>;
    public status: RaceControlData;
    private firstLoad: boolean = true;

    private track: string = "";
    private trackLayout: string = "";

    constructor() {
        this.$eventTitle = $("#event-title");
        this.status = new RaceControlData();

        if (!this.$eventTitle.length) {
            return;
        }

        let ws = new ReconnectingWebSocket(((window.location.protocol === "https:") ? "wss://" : "ws://") + window.location.host + "/api/race-control", [], {
            minReconnectionDelay: 0,
        });

        ws.onmessage = this.handleWebsocketMessage.bind(this);

        $(window).on('beforeunload', () => {
            ws.close();
        });

        this.handleIFrames();
        setInterval(this.showEventCompletion.bind(this), 1000);
        this.$eventTitle.on("click", function(e: ClickEvent) {
            e.preventDefault();
        });
    }

    private handleWebsocketMessage(ev: MessageEvent): void {
        let message = JSON.parse(ev.data) as WSMessage;

        if (!message) {
            return;
        }

        switch (message.EventType) {
            case EventVersion:
                location.reload();
                return;
            case EventRaceControl:
                this.status = new RaceControlData(message.Message);

                if (this.status.SessionInfo.Track !== this.track || this.status.SessionInfo.TrackConfig !== this.trackLayout) {
                    this.track = this.status.SessionInfo.Track;
                    this.trackLayout = this.status.SessionInfo.TrackConfig;
                    this.liveMap.onTrackChange(this.track, this.trackLayout);
                    this.liveTimings.onTrackChange(this.track, this.trackLayout);
                    this.onTrackChange(this.track, this.trackLayout);
                }

                this.$eventTitle.text(RaceControl.getSessionType(this.status.SessionInfo.Type) + " at " + this.status.TrackInfo!.name);
                $("#track-location").text(this.status.TrackInfo.city + ", " + this.status.TrackInfo.country);

                this.buildSessionInfo();

                if (this.firstLoad) {
                    this.showTrackWeatherImage();
                }

                this.firstLoad = false;
                break;
            case EventNewSession:
                this.showTrackWeatherImage();
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

    private showEventCompletion() {
        let timeRemaining = "";

        // Get lap/laps or time/totalTime
        if (this.status.SessionInfo.Time > 0) {
            timeRemaining = msToTime(this.status.SessionInfo.Time * 60 * 1000 - moment.duration(moment().diff(this.status.SessionStartTime)).asMilliseconds(), false, false);
        } else if (this.status.SessionInfo.Laps > 0) {
            let lapsCompleted = 0;

            if (this.status.ConnectedDrivers && this.status.ConnectedDrivers.GUIDsInPositionalOrder.length > 0) {
                let driver = this.status.ConnectedDrivers.Drivers[this.status.ConnectedDrivers.GUIDsInPositionalOrder[0]];

                if (driver.TotalNumLaps > 0) {
                    lapsCompleted = driver.TotalNumLaps;
                }
            }

            timeRemaining = this.status.SessionInfo.Laps - lapsCompleted + " laps remaining";
        }

        let $raceTime = $("#race-time");
        $raceTime.text(timeRemaining);
    }

    public onTrackChange(track: string, layout: string): void {
        $("#trackImage").attr("src", this.getTrackImageURL());

        $("#track-description").text(this.status.TrackInfo.description);
        $("#track-length").text(this.status.TrackInfo["length"]);
        $("#track-pitboxes").text(this.status.TrackInfo.pitboxes);
        $("#track-width").text(this.status.TrackInfo.width);
        $("#track-run").text(this.status.TrackInfo.run);
    }

    private buildSessionInfo() {
        let $roadTempWrapper = $("#road-temp-wrapper");
        $roadTempWrapper.attr("style", "background-color: " + getColorForPercentage(this.status.SessionInfo.RoadTemp / 40));
        $roadTempWrapper.attr("data-original-title", "Road Temp: " + this.status.SessionInfo.RoadTemp + "°C");

        let $roadTempText = $("#road-temp-text");
        $roadTempText.text(this.status.SessionInfo.RoadTemp + "°C");

        let $ambientTempWrapper = $("#ambient-temp-wrapper");
        $ambientTempWrapper.attr("style", "background-color: " + getColorForPercentage(this.status.SessionInfo.AmbientTemp / 40));
        $ambientTempWrapper.attr("data-original-title", "Ambient Temp: " + this.status.SessionInfo.AmbientTemp + "°C");

        let $ambientTempText = $("#ambient-temp-text");
        $ambientTempText.text(this.status.SessionInfo.AmbientTemp + "°C");

        $("#event-name").text(this.status.SessionInfo.Name);
        $("#event-type").text(RaceControl.getSessionType(this.status.SessionInfo.Type));
    }

    private showTrackWeatherImage(): void {
        let $currentWeather = $("#weatherImage");

        // Fix for sol weathers with time info in this format:
        // sol_05_Broken%20Clouds_type=18_time=0_mult=20_start=1551792960/preview.jpg
        let pathCorrected = this.status.SessionInfo.WeatherGraphics.split("_");

        for (let i = 0; i < pathCorrected.length; i++) {
            if (pathCorrected[i].indexOf("type=") !== -1) {
                pathCorrected.splice(i);
                break;
            }
        }

        let pathFinal = pathCorrected.join("_");

        $.get("/content/weather/" + pathFinal + "/preview.jpg").done(function () {
            // preview for skin exists
            $currentWeather.attr("src", "/content/weather/" + pathFinal + "/preview.jpg").show();
        }).fail(function () {
            // preview doesn't exist, load default fall back image
            $currentWeather.hide();
        });

        $currentWeather.attr("alt", "Current Weather: " + prettifyName(this.status.SessionInfo.WeatherGraphics, false));
    }

    private getTrackImageURL(): string {
        if (!this.status) {
            return "";
        }

        const sessionInfo = this.status.SessionInfo;

        return "/content/tracks/" + sessionInfo.Track + "/ui" + (!!sessionInfo.TrackConfig ? "/" + sessionInfo.TrackConfig : "") + "/preview.png";
    }

    private handleIFrames(): void {
        const $document = $(document);

        $document.on("change", ".live-frame-link", function (e: ChangeEvent) {
            let $this = $(e.currentTarget) as JQuery<HTMLInputElement>;
            let value = $this.val() as string;

            if (value) {
                let $liveTimingFrame = $this.closest(".live-frame-wrapper").find(".live-frame");
                $this.closest(".live-frame-wrapper").find(".embed-responsive").attr("class", "embed-responsive embed-responsive-16by9");

                // if somebody pasted an embed code just grab the actual link
                if (value.startsWith('<iframe')) {
                    let res = value.split('"');

                    for (let i = 0; i < res.length; i++) {
                        if (res[i] === " src=") {
                            if (res[i + 1]) {
                                $liveTimingFrame.attr("src", res[i + 1]);
                            }

                            $this.val(res[i + 1]);
                        }
                    }
                } else {
                    $liveTimingFrame.attr("src", value);
                }
            }
        });

        $document.on("click", ".remove-live-frame", function (e: ClickEvent) {
            $(e.currentTarget).closest(".live-frame-wrapper").remove();
        });

        $document.find("#add-live-frame").click(function () {
            let $copy = $document.find(".live-frame-wrapper").first().clone();

            $copy.removeClass("d-none");
            $copy.find(".embed-responsive").attr("class", "d-none embed-responsive embed-responsive-16by9");
            $copy.find(".frame-input").removeClass("ml-0");

            $document.find(".live-frame-wrapper").last().after($copy);
        });
    }
}

class LiveMap implements WebsocketHandler {
    private mapImageHasLoaded: boolean = false;

    private readonly $map: JQuery<HTMLDivElement>;
    private readonly $trackMapImage: JQuery<HTMLImageElement>;
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

                for (const connectedGUID in this.raceControl.status.ConnectedDrivers!.Drivers) {
                    const driver = this.raceControl.status.ConnectedDrivers!.Drivers[connectedGUID];

                    if (!this.dots.has(driver.CarInfo.DriverGUID)) {
                        // in the event that a user just loaded the race control page, place the
                        // already loaded dots onto the map
                        let $driverDot = this.buildDriverDot(driver.CarInfo, driver.LastPos as CarUpdateVec).show();
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
                    'background': randomColorForDriver(driverGUID),
                });

                $rpmGaugeOuter.append($rpmGaugeInner);
                $myDot!.find(".info").text(speed + "Km/h " + (update.Gear - 1));
                $myDot!.find(".info").append($rpmGaugeOuter);
                break;

            case EventNewSession:
                this.loadTrackMapImage();

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

    public onTrackChange(track: string, trackLayout: string): void {
        this.loadTrackMapImage();
    }

    private translateToTrackCoordinate(vec: CarUpdateVec): CarUpdateVec {
        const out = new CarUpdateVec();

        out.X = ((vec.X + this.trackXOffset + this.trackMargin) / this.trackScale) * this.mapScaleMultiplier;
        out.Z = ((vec.Z + this.trackZOffset + this.trackMargin) / this.trackScale) * this.mapScaleMultiplier;

        return out;
    }

    private buildDriverDot(driverData: SessionCarInfo, lastPos?: CarUpdateVec): JQuery<HTMLElement> {
        if (this.dots.has(driverData.DriverGUID)) {
            return this.dots.get(driverData.DriverGUID)!;
        }

        const $driverName = $("<span class='name'/>").text(driverData.DriverInitials);
        const $info = $("<span class='info'/>").text("0").hide();

        const $dot = $("<div class='dot' style='background: " + randomColorForDriver(driverData.DriverGUID) + "'/>").append($driverName, $info).hide().appendTo(this.$map);

        if (lastPos !== undefined) {
            let dotPos = this.translateToTrackCoordinate(lastPos);

            $dot.css({
                "left": dotPos.X,
                "top": dotPos.Z,
            });
        }

        this.dots.set(driverData.DriverGUID, $dot);

        return $dot;
    }

    private getTrackMapURL(): string {
        if (!this.raceControl.status) {
            return "";
        }

        const sessionInfo = this.raceControl.status.SessionInfo;

        return "/content/tracks/" + sessionInfo.Track + (!!sessionInfo.TrackConfig ? "/" + sessionInfo.TrackConfig : "") + "/map.png";
    }

    private loadTrackMapImage(): void {
        const trackURL = this.getTrackMapURL();
        let that = this;

        this.$trackMapImage.on("load", function() {
            that.mapImageHasLoaded = true;
            that.correctMapDimensions();
        });

        this.$trackMapImage.attr({"src": trackURL});
    }

    private static mapRotationRatio: number = 1.07;

    private correctMapDimensions(): void {
        if (!this.$trackMapImage || !this.mapImageHasLoaded) {
            return;
        }

        if (this.$trackMapImage.height()! / this.$trackMapImage.width()! > LiveMap.mapRotationRatio) {
            // rotate the map
            this.$map.addClass("rotated");

            this.$trackMapImage.css({
                'max-height': this.$trackMapImage.closest(".map-container").width()!,
                'max-width': 'auto'
            });

            this.mapScaleMultiplier = this.$trackMapImage.width()! / this.raceControl.status.TrackMapData.width;

            this.$map.closest(".map-container").css({
                'max-height': (this.raceControl.status.TrackMapData.width * this.mapScaleMultiplier) + 20,
            });

            this.$map.css({
                'max-width': (this.raceControl.status.TrackMapData.width * this.mapScaleMultiplier) + 20,
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

            this.mapScaleMultiplier = this.$trackMapImage.width()! / this.raceControl.status.TrackMapData.width;
        }
    }

    public getDotForDriverGUID(guid: string): JQuery<HTMLElement> | undefined {
        return this.dots.get(guid);
    }
}

const DriverGUIDDataKey = "driver-guid";

enum SessionType {
    Race = 3,
    Qualifying = 2,
    Practice = 1,
    Booking = 0,
}

enum Collision {
    WithCar = "with other car",
    WithEnvironment = "with environment",
}

class LiveTimings implements WebsocketHandler {
    private readonly raceControl: RaceControl;
    private readonly liveMap: LiveMap;

    private readonly $connectedDriversTable: JQuery<HTMLTableElement>;
    private readonly $disconnectedDriversTable: JQuery<HTMLTableElement>;
    private readonly $storedTimes: JQuery<HTMLDivElement>;

    constructor(raceControl: RaceControl, liveMap: LiveMap) {
        this.raceControl = raceControl;
        this.liveMap = liveMap;
        this.$connectedDriversTable = $("#live-table");
        this.$disconnectedDriversTable = $("#live-table-disconnected");
        this.$storedTimes = $("#stored-times");

        setInterval(this.populateConnectedDrivers.bind(this), 1000);

        $(document).on("click", ".driver-link", this.toggleDriverSpeed.bind(this));
    }

    public handleWebsocketMessage(message: WSMessage): void {
        if (message.EventType === EventRaceControl) {
            this.populateConnectedDrivers();
            this.populateDisconnectedDrivers();
        } else if (message.EventType === EventConnectionClosed) {
            const closedConnection = message.Message as SessionCarInfo;

            if (this.raceControl.status.ConnectedDrivers) {
                const driver = this.raceControl.status.ConnectedDrivers.Drivers[closedConnection.DriverGUID];

                if (driver && driver.LoadedTime.toString() === "0001-01-01T00:00:00Z") {
                    // a driver joined but never loaded. remove them from the connected drivers table.
                    this.$connectedDriversTable.find("tr[data-guid='" + closedConnection.DriverGUID + "']").remove();
                }
            }
        }
    }

    public onTrackChange(track: string, trackLayout: string): void {

    }

    private populateConnectedDrivers(): void {
        if (!this.raceControl.status || !this.raceControl.status.ConnectedDrivers) {
            return;
        }

        for (const driverGUID of this.raceControl.status.ConnectedDrivers.GUIDsInPositionalOrder) {
            const driver = this.raceControl.status.ConnectedDrivers.Drivers[driverGUID];

            if (!driver) {
                continue;
            }

            this.addDriverToTable(driver, this.$connectedDriversTable);
            this.populatePreviousLapsForDriver(driver);
        }
    }

    private populatePreviousLapsForDriver(driver: Driver): void {
        for (const carName in driver.Cars) {
            if (carName === driver.CarInfo.CarModel) {
                continue;
            }

            // create a fake new driver from the old driver. override details with their previous car
            // and add them to the disconnected drivers table. if the user rejoins in this car it will
            // be removed from the disconnected drivers table and placed into the connected drivers table.
            const dummyDriver = JSON.parse(JSON.stringify(driver));
            dummyDriver.CarInfo.CarModel = carName;

            this.addDriverToTable(dummyDriver, this.$disconnectedDriversTable);
        }
    }

    private populateDisconnectedDrivers(): void {
        if (!this.raceControl.status || !this.raceControl.status.DisconnectedDrivers) {
            return;
        }

        for (const driverGUID of this.raceControl.status.DisconnectedDrivers.GUIDsInPositionalOrder) {
            const driver = this.raceControl.status.DisconnectedDrivers.Drivers[driverGUID];

            if (!driver) {
                continue;
            }

            this.addDriverToTable(driver, this.$disconnectedDriversTable);
            this.populatePreviousLapsForDriver(driver);
        }

        if (this.$disconnectedDriversTable.find("tr").length > 1) {
            this.$storedTimes.show();
        } else {
            this.$storedTimes.hide();
        }
    }

    private addDriverToTable(driver: Driver, $table: JQuery<HTMLTableElement>): void {
        const addingDriverToConnectedTable = ($table === this.$connectedDriversTable);
        const carInfo = driver.Cars[driver.CarInfo.CarModel];

        if (!carInfo) {
            return;
        }

        const $tr = $("<tr class='driver-row'/>").attr({
            "data-guid": driver.CarInfo.DriverGUID,
            "data-car-model": driver.CarInfo.CarModel,
        });

        // car position
        if (addingDriverToConnectedTable) {
            const $tdPos = $("<td class='text-center'/>").text(driver.Position === 255 || driver.Position === 0 ? "" : driver.Position);
            $tr.append($tdPos);
        }

        // driver name
        const $tdName = $("<td/>").text(driver.CarInfo.DriverName);

        if (addingDriverToConnectedTable) {
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

        if (addingDriverToConnectedTable) {
            let currentLapTimeText = "";

            if (moment(carInfo.LastLapCompletedTime).isAfter(moment(this.raceControl.status!.SessionStartTime))) {
                // only show current lap time text if the last lap completed time is after session start.
                currentLapTimeText = msToTime(moment().diff(moment(carInfo.LastLapCompletedTime)), false);
            }

            const $tdCurrentLapTime = $("<td/>").text(currentLapTimeText);
            $tr.append($tdCurrentLapTime);
        }

        if (addingDriverToConnectedTable) {
            // last lap
            const $tdLastLap = $("<td/>").text(msToTime(carInfo.LastLap / 1000000));
            $tr.append($tdLastLap);
        }

        // best lap
        const $tdBestLapTime = $("<td/>").text(msToTime(carInfo.BestLap / 1000000));
        $tr.append($tdBestLapTime);

        if (addingDriverToConnectedTable) {
            // gap
            const $tdGap = $("<td/>").text(driver.Split);
            $tr.append($tdGap);
        }

        // lap number
        const $tdLapNum = $("<td/>").text(carInfo.NumLaps ? carInfo.NumLaps : "0");
        $tr.append($tdLapNum);

        const $tdTopSpeedBestLap = $("<td/>").text(carInfo.TopSpeedBestLap ? carInfo.TopSpeedBestLap.toFixed(2) + "Km/h" : "");
        $tr.append($tdTopSpeedBestLap);

        if (addingDriverToConnectedTable) {
            // events
            const $tdEvents = $("<td/>");

            if (moment(driver.LoadedTime).add("10", "seconds").isSameOrAfter(moment())) {
                // car just loaded
                let $tag = $("<span/>");
                $tag.attr({'class': 'badge badge-success live-badge'});
                $tag.text("Loaded");

                $tdEvents.append($tag);
            }

            if (driver.Collisions) {
                for (const collision of driver.Collisions) {
                    if (moment(collision.Time).add("10", "seconds").isSameOrAfter(moment())) {
                        let $tag = $("<span/>");
                        $tag.attr({'class': 'badge badge-danger live-badge'});

                        if (collision.Type === Collision.WithCar) {
                            $tag.text(
                                "Crash with " + collision.OtherDriverName + " at " + collision.Speed.toFixed(2) + "Km/h"
                            );
                        } else {
                            $tag.text(
                                "Crash " + collision.Type + " at " + collision.Speed.toFixed(2) + "Km/h"
                            );
                        }

                        $tdEvents.append($tag);
                    }
                }
            }

            $tr.append($tdEvents);
        }

        // remove this driver and car from our current table.
        $table.find("[data-guid='" + driver.CarInfo.DriverGUID + "'][data-car-model='" + driver.CarInfo.CarModel + "']").remove();

        if (!addingDriverToConnectedTable) {
            // if we're adding to the disconnected table, ensure we've removed this driver and car from the connected table.
            this.$connectedDriversTable.find("[data-guid='" + driver.CarInfo.DriverGUID + "'][data-car-model='" + driver.CarInfo.CarModel + "']").remove();
        } else {
            // remove the driver from the disconnected table
            this.$disconnectedDriversTable.find("[data-guid='" + driver.CarInfo.DriverGUID + "'][data-car-model='" + driver.CarInfo.CarModel + "']").remove();
        }

        if (!addingDriverToConnectedTable && (!carInfo.NumLaps || carInfo.NumLaps === 0)) {
            return;
        }

        $table.append($tr);

        if (!addingDriverToConnectedTable) {
            this.sortTable($table);
        }
    }

    private sortTable($table: JQuery<HTMLTableElement>) {
        const $tbody = $table.find("tbody");
        const that = this;

        $($tbody.find("tr:not(:nth-child(1))").get().sort(function (a: HTMLTableElement, b: HTMLTableElement): number {
            if (that.raceControl.status.SessionInfo.Type == SessionType.Race) {
                let lapsA = parseInt($(a).find("td:nth-child(4)").text(), 10);
                let lapsB = parseInt($(b).find("td:nth-child(4)").text(), 10);

                if (lapsA !== 0 && lapsB !== 0 && lapsA < lapsB) {
                    return 1;
                } else if (lapsA === lapsB) {
                    return 0;
                } else {
                    return -1;
                }
            } else {
                let timeA = $(a).find("td:nth-child(3)").text();
                let timeB = $(b).find("td:nth-child(3)").text();

                if (timeA !== "" && timeB !== "" && timeA < timeB) {
                    return -1;
                } else if (timeA === timeB) {
                    return 0;
                } else {
                    return 1;
                }
            }
        })).appendTo($tbody);
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

function randomColorForDriver(driverGUID: string): string {
    return randomColor({
        seed: driverGUID,
    })
}

const percentColors = [
    {pct: 0.25, color: {r: 0x00, g: 0x00, b: 0xff}},
    {pct: 0.625, color: {r: 0x00, g: 0xff, b: 0}},
    {pct: 1.0, color: {r: 0xff, g: 0x00, b: 0}}
];

function getColorForPercentage(pct: number): string {
    let i;

    for (i = 1; i < percentColors.length - 1; i++) {
        if (pct < percentColors[i].pct) {
            break;
        }
    }

    let lower = percentColors[i - 1];
    let upper = percentColors[i];
    let range = upper.pct - lower.pct;
    let rangePct = (pct - lower.pct) / range;
    let pctLower = 1 - rangePct;
    let pctUpper = rangePct;
    let color = {
        r: Math.floor(lower.color.r * pctLower + upper.color.r * pctUpper),
        g: Math.floor(lower.color.g * pctLower + upper.color.g * pctUpper),
        b: Math.floor(lower.color.b * pctLower + upper.color.b * pctUpper)
    };

    return 'rgb(' + [color.r, color.g, color.b].join(',') + ')';
}
