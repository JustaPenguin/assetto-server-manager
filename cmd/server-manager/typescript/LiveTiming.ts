import {timeDiff} from "./Utils";
import * as randomColor from "../node_modules/randomcolor/";

export class LiveTiming {
    private static readonly REFRESH_INTERVAL = 1000;

    private raceCompletion: string = "";
    private $liveTimingTable: JQuery<HTMLElement> | null = null;
    private total: string = "";
    private sessionType: string = "";

    public constructor() {
        let $liveTimingTable = $("#live-table");

        if ($liveTimingTable.length) {
            setInterval(() => {
                this.loadLiveTimings();
            }, LiveTiming.REFRESH_INTERVAL);
        }
    }

    private loadLiveTimings() {
        let that = this;

        $.getJSON("/live-timing/get", function (liveTiming) {
            let date = new Date();

            // Get lap/laps or time/totalTime
            if (liveTiming.Time > 0) {
                that.total = liveTiming.Time + "m";

                that.raceCompletion = timeDiff(new Date(liveTiming.SessionStarted), date);
            } else if (liveTiming.Laps > 0) {
                that.raceCompletion = liveTiming.LapNum;
                that.total = liveTiming.Laps + " laps";
            }

            let $raceTime = $("#race-time");
            $raceTime.text("Event Completion: " + that.raceCompletion + "/ " + that.total);

            // Get the session type
            let $currentSession = $("#current-session");

            switch (liveTiming.Type) {
                case 0:
                    that.sessionType = "Booking";
                    break;
                case 1:
                    that.sessionType = "Practice";
                    break;
                case 2:
                    that.sessionType = "Qualifying";
                    break;
                case 3:
                    that.sessionType = "Race";
                    break;
            }

            $currentSession.text("Current Session: " + that.sessionType);

            for (let car in liveTiming.Cars) {
                if (liveTiming.Cars[car].Pos === 0) {
                    liveTiming.Cars[car].Pos = 255
                }
            }

            // Get active cars - sort by pos
            let sorted = Object.keys(liveTiming.Cars)
                .sort((a: string, b: string): number => {
                    if (liveTiming.Cars[a].Pos < liveTiming.Cars[b].Pos) {
                        return -1
                    } else if (liveTiming.Cars[a].Pos === liveTiming.Cars[b].Pos) {
                        return 0
                    } else if (liveTiming.Cars[a].Pos > liveTiming.Cars[b].Pos) {
                        return 1
                    }

                    return 0;
                });

            for (let car of sorted) {
                let $driverRow = $("#" + liveTiming.Cars[car].DriverGUID);
                let $tr;

                let lapTime = "";

                // Get the lap time, display previous for 10 seconds after completion
                if (liveTiming.Cars[car].LastLapCompleteTimeUnix + 10000 > date.getTime()) {
                    lapTime = liveTiming.Cars[car].LastLap
                } else if (liveTiming.Cars[car].LapNum === 0) {
                    lapTime = "0s"
                } else {
                    lapTime = timeDiff(liveTiming.Cars[car].LastLapCompleteTimeUnix, date)
                }

                if ($driverRow.length) {
                    $driverRow.remove()
                }

                $tr = $("<tr/>");
                $tr.attr({'id': liveTiming.Cars[car].DriverGUID});
                $tr.empty();

                let $tdPos = $("<td/>");
                let $tdName = $("<td/>");
                let $tdLapTime = $("<td/>");
                let $tdBestLap = $("<td/>");
                let $tdGap = $("<td/>");
                let $tdLapNum = $("<td/>");
                let $tdEvents = $("<td/>");

                if (liveTiming.Cars[car].Pos === 255) {
                    $tdPos.text("n/a");
                } else {
                    $tdPos.text(liveTiming.Cars[car].Pos);
                }
                $tr.append($tdPos);

                $tdName.text(liveTiming.Cars[car].DriverName);
                $tdName.prepend($("<div class='dot' style='background: " + randomColor({
                    luminosity: 'bright',
                    seed: liveTiming.Cars[car].DriverGUID
                }) + "'/>"));
                $tr.append($tdName);

                $tdLapTime.text(lapTime);
                $tr.append($tdLapTime);

                $tdBestLap.text(liveTiming.Cars[car].BestLap);
                $tr.append($tdBestLap);

                $tdGap.text(liveTiming.Cars[car].Split);
                $tr.append($tdGap);

                $tdLapNum.text(liveTiming.Cars[car].LapNum);
                $tr.append($tdLapNum);

                if (liveTiming.Cars[car].Loaded && liveTiming.Cars[car].LoadedTime + 10000 > date.getTime()) {
                    let $tag = $("<span/>");
                    $tag.attr({'class': 'badge badge-success live-badge'});
                    $tag.text("Loaded");

                    $tdEvents.append($tag);
                }

                if (liveTiming.Cars[car].Collisions !== null) {
                    for (let y = 0; y < liveTiming.Cars[car].Collisions.length; y++) {
                        if (liveTiming.Cars[car].Collisions[y].Time + 10000 > date.getTime()) {
                            let $tag = $("<span/>");
                            $tag.attr({'class': 'badge badge-danger live-badge'});
                            $tag.text("Crash " + liveTiming.Cars[car].Collisions[y].Type + " at " +
                                parseFloat(liveTiming.Cars[car].Collisions[y].Speed).toFixed(2) + "m/s");

                            $tdEvents.append($tag);
                        }
                    }
                }

                $tr.append($tdEvents);

                if (that.$liveTimingTable) {
                    that.$liveTimingTable.append($tr)
                }
            }
        });
    }
}