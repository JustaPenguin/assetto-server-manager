import {RRule} from "rrule"
import ClickEvent = JQuery.ClickEvent;


export class RaceList {
    public constructor() {
        this.initRecurrenceRuleExplanations();
        this.initRaceDetailsButtons();
    }

    private initRecurrenceRuleExplanations(): void {
        let rRules = document.getElementsByClassName('rrule-text');

        for (let i = 0; i < rRules.length; i++) {
            let recurrenceString = rRules[i].getAttribute("data-rrule");

            if (recurrenceString) {
                let rule = RRule.fromString(recurrenceString);

                rRules[i].textContent = rule.toText()
            }
        }
    }

    private initRaceDetailsButtons(): void {
        $(document).on("click", ".custom-race-details", (e: ClickEvent) => {
            let $this = $(e.currentTarget);
            let raceID = $this.attr("data-race-id");

            const modalContentURL = `/event-details?custom-race=${raceID}`;

            $.get(modalContentURL).then((data: string) => {
                let $eventDetailsModal = $("#race-details-modal");
                $eventDetailsModal.html(data);
                $eventDetailsModal.find("input[type='checkbox']").bootstrapSwitch();
                $eventDetailsModal.modal();
            });

            return false;
        });
    }
}
