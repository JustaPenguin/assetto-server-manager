import ChangeEvent = JQuery.ChangeEvent;

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

            console.log(val);

            this.$raceWeekendSession.find(".sessions .tab-pane").removeClass(["show", "active"]);
            this.$raceWeekendSession.find(".session-enabler").each(function (index, elem) {
                console.log($(elem));

                $(elem).bootstrapSwitch("state", false);
            });

            this.$raceWeekendSession.find("#session-" + val).addClass(["show", "active"]);
            this.$raceWeekendSession.find("#" + val + ".Enabled").bootstrapSwitch("setState", true);
        });
    }
}