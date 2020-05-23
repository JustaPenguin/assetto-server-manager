import dragula from "dragula";
import {RaceSetup} from "./RaceSetup";
import {ordinalSuffix, prettifyName} from "./utils";

declare var ChampionshipID: string;
declare var CanMoveChampionshipEvents: boolean;
declare var availableCars: any;
declare var defaultPoints: any;

export namespace Championship {
    export class View {
        private $document = $(document);

        public constructor() {
            this.initDraggableCards();
            this.initDisplayOrder();
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

            $(".championship-event").each(function () {
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

        private initDisplayOrder() {
            let $hideCompleted = this.$document.find("#hide-completed");
            let $hideNotCompleted = this.$document.find("#hide-not-completed");
            let $switchOrder = this.$document.find("#switch-order");


            let $championship = this.$document.find(".championship").first();

            let that = this;

            $hideCompleted.click(function (e) {
                e.preventDefault();

                let $events = that.$document.find(".championship-event");

                for (let i = 0; i < $events.length; i++) {
                    if ($($events[i]).hasClass("event-complete")) {
                        if ($($events[i]).is(":hidden")) {
                            $($events[i]).show();
                            $(this).attr("class", "dropdown-item text-success")
                        } else {
                            $($events[i]).hide();
                            $(this).attr("class", "dropdown-item text-danger")
                        }
                    }
                }
            });

            $hideNotCompleted.click(function (e) {
                e.preventDefault();

                let $events = that.$document.find(".championship-event");

                for (let i = 0; i < $events.length; i++) {
                    if (!$($events[i]).hasClass("event-complete")) {
                        if ($($events[i]).is(":hidden")) {
                            $($events[i]).show();
                            $(this).attr("class", "dropdown-item text-success")
                        } else {
                            $($events[i]).hide();
                            $(this).attr("class", "dropdown-item text-danger")
                        }
                    }
                }
            });

            $switchOrder.click(function (e) {
                e.preventDefault();

                let $events = that.$document.find(".championship-event");

                for (let i = $events.length; i >= 0; i--) {
                    if (i === $events.length) {
                        continue
                    }

                    $($events[i]).insertAfter($championship.find(".championship-event").last())
                }

                $(this).find(".fa-stack-1x").each(function () {
                    let icon;
                    if ($(this).hasClass("fa-sort-up")) {
                        icon = "fa-sort-up"
                    } else {
                        icon = "fa-sort-down"
                    }

                    if ($(this).hasClass("text-success")) {
                        $(this).attr("class", icon + " fas fa-stack-1x text-dark")
                    } else {
                        $(this).attr("class", icon + " fas fa-stack-1x text-success")
                    }
                })
            });
        }
    }

    export class Edit {
        private $document = $(document);
        private $pointsTemplate: JQuery<Document>;

        public constructor() {
            this.$pointsTemplate = this.$document.find(".points-place").last().clone();

            $(".race-setup").each(function (index, elem) {
                // init totalNumPoints val to be equal to the number of .points-place's visible in the class.
                let $raceSetup = $(elem);
                let $savedNumPoints = $raceSetup.find(".totalNumPoints");
                $savedNumPoints.val($raceSetup.find(".points-place:visible").length);
            });

            this.$document.on("click", ".addEntrant", function (e) {

            });

            this.initClassSetup();
            this.initSummerNote();
            this.initConfigureSignUpForm();
            this.initACSRWatcher();
        }

        private initACSRWatcher() {
            let $acsrSwitch = $("#ACSR");

            if ($acsrSwitch.length) {
                let state = $acsrSwitch.bootstrapSwitch('state');

                this.setSwitchesForACSR(state);
            }

            $acsrSwitch.on('switchChange.bootstrapSwitch', (event, state) => {
                this.setSwitchesForACSR(state);
            });
        }

        private setSwitchesForACSR(state) {
            let $openEntrantsSwitch = $("#ChampionshipOpenEntrants");
            let $signUpFormSwitch = $("#ChampionshipSignUpFormEnabled");
            let $overridePasswordSwitch = $("#OverridePassword");

            if (state) {
                $openEntrantsSwitch.bootstrapSwitch('state', false);
                $openEntrantsSwitch.bootstrapSwitch('disabled', true);

                $signUpFormSwitch.bootstrapSwitch('state', true);
                $signUpFormSwitch.bootstrapSwitch('disabled', true);

                $overridePasswordSwitch.bootstrapSwitch('state', true);
                $overridePasswordSwitch.bootstrapSwitch('disabled', true);

                $overridePasswordSwitch.closest(".card-body").find("#ReplacementPasswordWrapper").hide();
            } else {
                $overridePasswordSwitch.closest(".card-body").find("#ReplacementPasswordWrapper").show();

                $openEntrantsSwitch.bootstrapSwitch('disabled', false);
                $signUpFormSwitch.bootstrapSwitch('disabled', false);
                $overridePasswordSwitch.bootstrapSwitch('disabled', false);
            }
        }

        private initConfigureSignUpForm() {
            let $showWhenSignUpEnabled = $(".show-signup-form-enabled");

            $("#ChampionshipSignUpFormEnabled").on('switchChange.bootstrapSwitch', function (event, state) {
                if (state) {
                    $showWhenSignUpEnabled.show();
                } else {
                    $showWhenSignUpEnabled.hide();
                }
            });

            let $clonedQuestion = $(".championship-signup-question").first().clone();

            $("#AddSignUpFormQuestion").on("click", function (e) {
                e.preventDefault();

                let $newQuestion = $clonedQuestion.clone();
                $newQuestion.find("input").val("");
                $newQuestion.appendTo($("#Questions"));
            });

            this.$document.on("click", ".btn-delete-question", function (e) {
                e.preventDefault();

                $(this).closest(".championship-signup-question").remove();
            })
        }

        private initClassSetup() {
            let $addClassButton = this.$document.find("#addClass");
            let $tmpl = this.$document.find("#class-template");
            let $classTemplate = $tmpl.clone() as unknown as JQuery<HTMLElement>;

            $tmpl.remove();


            $addClassButton.click(function (e) {
                e.preventDefault();

                let $cloned = $classTemplate.clone().show();

                $(this).before($cloned);
                new RaceSetup($cloned);

                let maxClients = $("#MaxClients").val() as number;

                if ($(".entrant:visible").length >= maxClients) {
                    $cloned.find(".entrant:visible").remove();
                }
            });

            this.$document.on("click", ".btn-delete-class", function (e) {
                e.preventDefault();
                $(this).closest(".race-setup").remove();
            });

            this.$document.on("input", ".entrant-team", function () {
                $(this).closest(".entrant").find(".points-transfer").show();
            });

            this.$document.on("change", ".Cars", function (e) {
                let $target = $(e.currentTarget);

                $target.closest(".race-setup").find("input[name='NumCars']").val(($target.val() as string).length);
            });
        }

        private initSummerNote() {
            if ($(".championship").length === 0 && $("#championship-form").length === 0) {
                return;
            }

            let $summerNote = this.$document.find("#summernote");
            let $ChampionshipInfoHolder = this.$document.find("#ChampionshipInfoHolder");

            if ($ChampionshipInfoHolder.length > 0) {
                $summerNote.summernote('code', $ChampionshipInfoHolder.html());
            }

            $summerNote.summernote({
                placeholder: 'You can use this text input to share links to tracks/cars or any other resources, outline' +
                    ' Championship rules and anything else you can think of with the entrants of your championship!' +
                    ' You can just leave this blank if you don\'t want any info! Large images will degrade the load time' +
                    ' of this edit championship page, it shouldn\'t affect the view page too much though.',
                tabsize: 2,
                height: 200,
            });
        }

        private addEntrant(e: JQuery.ClickEvent): void {
            e.preventDefault();

            let $raceSetup = $(this).closest(".race-setup");
            let $pointsParent = $raceSetup.find(".points-parent");

            if (!$pointsParent.length) {
                return;
            }

            let $points = $raceSetup.find(".points-place");
            let numEntrants = $raceSetup.find(".entrant:visible").length;
            let numPoints = $points.length;

            for (let i = numPoints; i < numEntrants; i++) {
                // add points up to the numEntrants we have
                let $newPoints = this.$pointsTemplate.clone();
                $newPoints.find("label").text(ordinalSuffix(i + 1) + " Place");

                let pointsVal = 0;

                // load the default points value for this position
                if (i < defaultPoints.Places.length) {
                    pointsVal = defaultPoints.Places[i];
                }

                $newPoints.find("input").attr({"value": pointsVal});
                $pointsParent.append($newPoints);
            }

            let $savedNumPoints = $raceSetup.find(".totalNumPoints");
            $savedNumPoints.val($raceSetup.find(".points-place:visible").length);
        }
    }

    export class SignUpForm {
        private $skinsDropdown: JQuery<HTMLElement>;
        private $carsDropdown: JQuery<HTMLElement>;
        private $carPreviewImage: JQuery<HTMLElement>;

        public constructor() {
            let $signUpForm = $("#championship-signup-form");
            this.$skinsDropdown = $signUpForm.find("#Skin");
            this.$carsDropdown = $signUpForm.find("#Car");
            this.$carPreviewImage = $signUpForm.find("#CarPreview");

            if ($signUpForm.length < 1) {
                return;
            }

            this.populateSkinsDropdown(this.$carsDropdown.val() as string);
            this.showCarImage(this.$carsDropdown.val() as string, this.$skinsDropdown.val() as string);

            this.$carsDropdown.on("change", (e: JQuery.ChangeEvent) => {
                let $this = $(e.currentTarget);
                this.populateSkinsDropdown($this.val() as string);
                this.showCarImage($this.val() as string, this.$skinsDropdown.val() as string);
            });

            this.$skinsDropdown.on("change", () => {
                this.showCarImage(this.$carsDropdown.val() as string, this.$skinsDropdown.val() as string);
            });
        }


        private populateSkinsDropdown(car: string): void {
            if (typeof availableCars === "undefined") {
                return;
            }

            let selected = this.$skinsDropdown.val();

            this.$skinsDropdown.empty();

            if (car in availableCars) {
                for (let skin of availableCars[car]) {
                    this.$skinsDropdown.append($("<option>", {
                        "val": skin,
                        "text": prettifyName(skin, true),
                        "selected": skin === selected,
                    }));
                }
            }
        }

        private showCarImage(car: string, skin: string): void {
            let path = "/content/cars/" + car + "/skins/" + skin + "/preview.jpg";
            let that = this;

            $.get(path)
                .done(function () {
                    that.$carPreviewImage.attr({"src": path, "alt": prettifyName(skin, false)})
                })
                .fail(function () {
                    path = "/static/img/no-preview-car.png";
                    that.$carPreviewImage.attr({"src": path, "alt": "Preview Image"})
                })
            ;
        }
    }
}
