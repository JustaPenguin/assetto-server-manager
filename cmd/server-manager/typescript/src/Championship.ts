import dragula from "dragula";
import ClickEvent = JQuery.ClickEvent;

declare var ChampionshipID: string;
declare var CanMoveChampionshipEvents: boolean;

interface ACSRDriverRatingGateMet {
    acsr_driver_rating: ACSRDriverRating;
    gate_met: boolean;
    acsr_enabled: boolean;
}

interface ACSRDriverRating {
    driver_id: number;
    skill_rating_grade: string;
    skill_rating: number;
    safety_rating: number;
    num_events: number;
    is_provisional: boolean;
}

export namespace Championship {

    export class View {
        public constructor() {
            this.initDraggableCards();
            this.initEventDetailsButtons();
            this.initACSRRatingWatcher();
        }

        private initDraggableCards(): void {
            let drake = dragula([document.querySelector(".championship-events")!], {
                moves: (el?: Element, source?: Element, handle?: Element, sibling?: Element): boolean => {
                    if (!CanMoveChampionshipEvents || !handle) {
                        return false;
                    }

                    return $(handle).hasClass("card-header");
                },
            });

            drake.on("drop", () => {
                this.saveChampionshipEventOrder();
            });
        }

        private saveChampionshipEventOrder(): void {
            let championshipEventIDs: string[] = [];

            $(".championship-event").each(function() {
                if (!$(this).hasClass("gu-mirror")) {
                    // dragula duplicates the element being moved as a 'mirror',
                    // ignore it when building championship event id list
                    championshipEventIDs.push($(this).attr("id")!);
                }
            });

            $.ajax({
                type: "POST",
                url: `/championship/${ChampionshipID}/reorder-events`,
                data: JSON.stringify(championshipEventIDs),
                dataType: "json"
            });
        }

        private initEventDetailsButtons(): void {
            $(document).on("click", ".championship-event-details", (e: ClickEvent) => {
                let $this = $(e.currentTarget);
                let eventID = $this.attr("data-event-id");

                const modalContentURL = `/event-details?championshipID=${ChampionshipID}&eventID=${eventID}`;

                $.get(modalContentURL).then((data: string) => {
                    let $eventDetailsModal = $("#event-details-modal");
                    $eventDetailsModal.html(data);
                    $eventDetailsModal.find("input[type='checkbox']").bootstrapSwitch();
                    $eventDetailsModal.modal();
                });

                return false;
            });
        }

        private initACSRRatingWatcher(): void {
            this.ACSRRatingWatcher();

            $(document).on("input", ".acsrGUID", (e: JQuery.Event) => {
                this.ACSRRatingWatcher();
            });
        }

        private ACSRRatingWatcher(): void {
            let val = $(".acsrGUID").val()
            if (val != undefined) {
                let valString = val.toString()

                let $registerButton = $("#register-for-championship");
                let $acsrRatingContainer = $("#acsr-rating-container");
                $acsrRatingContainer.empty();

                if (valString.length == 17) {
                    $.ajax({
                        type: "POST",
                        url: `/championship/${ChampionshipID}/${valString}/acsr-rating`,
                        data: "",
                        dataType: "json"
                    }).then((response: ACSRDriverRatingGateMet) => {
                        if (!response.acsr_enabled) {
                            return
                        }

                        if (response.acsr_driver_rating == null) {
                            let notFoundSpan = $('<span />');
                            let notFoundLink = $(`<a />`);

                            notFoundSpan.addClass("text-warning");
                            notFoundSpan.text("Sorry, we couldn't find an ACSR driver matching that GUID." +
                                " You can sign up for an account ");

                            notFoundLink.attr("href", "https://acsr.assettocorsaservers.com/")
                            notFoundLink.text("here!");

                            notFoundSpan.append(notFoundLink)
                            $acsrRatingContainer.append(notFoundSpan)

                            $registerButton.attr("disabled", "disabled");

                            return
                        }

                        let skillSpan = $('<span />');
                        let safetySpan = $('<span />');

                        skillSpan.addClass("badge badge-primary acsr-badge--skill mr-1");
                        skillSpan.text(response.acsr_driver_rating.skill_rating_grade);

                        safetySpan.addClass("badge badge-success acsr-badge--safety mr-1");
                        safetySpan.text(response.acsr_driver_rating.safety_rating);

                        $acsrRatingContainer.append(skillSpan);
                        $acsrRatingContainer.append(safetySpan);

                        let metSpan = $('<span />');

                        if (response.gate_met) {
                            metSpan.addClass("text-success");
                            metSpan.text("You meet the requirements!");

                            $registerButton.removeAttr("disabled");
                        } else {
                            metSpan.addClass("text-danger");
                            metSpan.text("Sorry, you don't meet the requirements!");

                            $registerButton.attr("disabled");
                        }

                        $acsrRatingContainer.append(metSpan);
                    })
                } else {
                    let incorrectGUIDSpan = $('<span />');
                    let incorrectGUIDLink = $(`<a />`);

                    incorrectGUIDSpan.addClass("text-warning");
                    incorrectGUIDSpan.text("It looks like your Steam GUID is not formatted correctly! Please enter " +
                        "your full GUID above now. You can find your GUID (steamID64) ");

                    incorrectGUIDLink.attr("href", "https://steamid.io/lookup")
                    incorrectGUIDLink.text("here!");

                    incorrectGUIDSpan.append(incorrectGUIDLink)
                    $acsrRatingContainer.append(incorrectGUIDSpan)

                    $registerButton.attr("disabled", "disabled");
                }
            }
        }
    }
}
