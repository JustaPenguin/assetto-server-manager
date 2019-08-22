import ChangeEvent = JQuery.ChangeEvent;

import {jsPlumb} from "jsplumb";

export class RaceWeekendSession {
    private $raceWeekendSession: JQuery<HTMLElement>;

    public constructor() {
        this.$raceWeekendSession = $("#race-weekend-session");

        if (!this.$raceWeekendSession.length) {
            return;
        }

        this.initSessionTypeSwitch();
    }

    private initSessionTypeSwitch(): void {
        const $sessionSwitcher = this.$raceWeekendSession.find("#SessionType");

        $sessionSwitcher.on("change", (e: ChangeEvent) => {
            const val = $(e.currentTarget).val();
            this.$raceWeekendSession.find(".sessions .tab-pane").removeClass(["show", "active"]);

            const $newSession = this.$raceWeekendSession.find("#session-" + val);
            $newSession.addClass(["show", "active"]);
            $newSession.find(".session-details").show();

            this.$raceWeekendSession.find(".session-enabler").prop("checked", false);
            this.$raceWeekendSession.find("#" + val + "\\.Enabled").prop("checked", true);
        });
    }
}

const plumb = jsPlumb.getInstance();

plumb.bind("ready", () => {
    console.log("PLUMBED");
});