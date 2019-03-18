import $ from "jquery";
import {prettifyName} from "./Utils";
import ChangeEvent = JQuery.ChangeEvent;
import ClickEvent = JQuery.ClickEvent;
import "jquery-ui-bundle";

let $entrantTemplate: JQuery<HTMLElement> | null = null;

declare var possibleEntrants, availableCars: any;

export class EntryList {
    private $parent: JQuery<HTMLElement>;
    private $carsDropdown: JQuery<HTMLSelectElement>;
    private driverNames: Array<string> = [];

    $entrantsDiv: JQuery<HTMLElement> | null = null;

    public constructor($parent: JQuery<HTMLElement>, $carsDropdown: JQuery<HTMLSelectElement>) {
        this.$parent = $parent;
        this.$carsDropdown = $carsDropdown;

        this.$entrantsDiv = this.$parent.find("#entrants");

        if (!this.$entrantsDiv.length) {
            return;
        }

        // initialise entrantTemplate if it's null. this will only happen once so cloned race setups
        // have an entrant template to work from.
        let $tmpl = this.$parent.find("#entrantTemplate");

        if (!$entrantTemplate) {
            $entrantTemplate = $tmpl.prop("id", "").clone(true, true);
        }

        $tmpl.remove();

        if (possibleEntrants) {
            for (let entrant of possibleEntrants) {
                this.driverNames.push(entrant.Name);
            }
        }

        this.$parent.find(".entryListCar").change((e: ChangeEvent) => {
            this.onEntryListCarChange(e)
        });

        // When the skin is changed on all initially loaded cars
        this.$parent.on("change", ".entryListSkin", (e: ChangeEvent) => {
            let car = $(e.currentTarget).closest(".entrant").find(".entryListCar").val() as string;
            let skin = $(e.currentTarget).val() as string;

            this.showEntrantSkin(car, skin, $(e.currentTarget))
        });

        this.populateEntryListCars();

        this.$carsDropdown.change(() => {
            this.populateEntryListCars();
        });

        this.autoCompleteDrivers();


        this.$parent.on("click", ".btn-delete-entrant", EntryList.deleteEntrant);
        this.$parent.on("click", ".addEntrant", (e: ClickEvent) => {
            this.addEntrant(e)
        });
    }

    private onEntryListCarChange(e: ChangeEvent): void {
        let $this = $(e.currentTarget);
        let car = $this.val() as string;
        let skin = $this.closest(".entrant").find(".entryListSkin").val() as string;

        EntryList.populateEntryListSkins($this, car);

        // When the car is changed for an added entrant
        this.showEntrantSkin(car, skin, $this)
    }

    private showEntrantSkin(currentCar: string, skin: string, $parent: JQuery<HTMLElement>) {
        if (currentCar in availableCars && availableCars[currentCar] != null && availableCars[currentCar].length > 0) {
            if (skin === "random_skin") {
                skin = availableCars[currentCar][0]
            }

            let path = "/content/cars/" + currentCar + "/skins/" + skin + "/preview.jpg";
            let $preview = $parent.closest(".entrant").find(".entryListCarPreview");

            $.get(path)
                .done(function () {
                    // preview for skin exists
                    $preview.attr({"src": path, "alt": prettifyName(skin, false)})
                }).fail(function () {
                // preview doesn't exist, load default fall back image
                path = "/static/img/no-preview-car.png";
                $preview.attr({"src": path, "alt": "Preview Image"})
            });
        }
    }

    private static populateEntryListSkins($elem: JQuery<HTMLElement>, car: string): void {
        // populate skins
        let $skinsDropdown = $elem.closest(".entrant").find(".entryListSkin");
        let selected = $skinsDropdown.val();

        $skinsDropdown.empty();

        $("<option value='random_skin'>&lt;random skin&gt;</option>").appendTo($skinsDropdown);

        try {

            if (car in availableCars && availableCars[car] != null) {
                for (let skin of availableCars[car]) {
                    let $opt = $("<option/>");
                    $opt.attr({'value': skin});
                    $opt.text(prettifyName(skin, true));

                    if (skin === selected) {
                        $opt.attr({'selected': 'selected'});
                    }

                    $opt.appendTo($skinsDropdown);
                }
            }
        } catch (e) {
            console.error(e);
        }
    }

    private addEntrant(e: ClickEvent): void {
        e.preventDefault();

        if ($entrantTemplate === null) {
            return;
        }

        let $this = $(e.currentTarget);
        let $numEntrantsField = $(e.currentTarget).parent().find(".numEntrantsToAdd");
        let numEntrantsToAdd = 1;

        if ($numEntrantsField.length > 0) {
            numEntrantsToAdd = $numEntrantsField.val() as number;
        }

        for (let i = 0; i < numEntrantsToAdd; i++) {
            let $elem = $entrantTemplate.clone();
            $elem.find("input[type='checkbox']").bootstrapSwitch();
            $elem.insertBefore($this.parent());


            this.populateEntryListCars();
            $elem.css("display", "block");
        }

        let $savedNumEntrants = this.$parent.find(".totalNumEntrants");
        $savedNumEntrants.val(this.$parent.find(".entrant:visible").length);
    }

    private static deleteEntrant(e: ClickEvent): void {
        e.preventDefault();

        let $raceSetup = $(e.currentTarget).closest(".race-setup");

        let numEntrants = $raceSetup.find(".entrant:visible").length;
        let $points = $raceSetup.find(".points-place");
        let numPoints = $points.length;


        for (let i = numPoints; i >= numEntrants; i--) {
            // remove any extras we don't need
            $points.last().remove();
        }

        $(e.currentTarget).closest(".entrant").remove();

        let $savedNumEntrants = $raceSetup.find(".totalNumEntrants");
        $savedNumEntrants.val($raceSetup.find(".entrant:visible").length);
    }

    private populateEntryListCars() {
        // populate list of cars in entry list
        let cars = new Set<string>(this.$carsDropdown.val() as Array<string>);

        let that = this;

        this.$parent.find(".entryListCar").each(function (index, val) {
            let $val = $(val);
            let selected = $val.find("option:selected").val() as string;

            if (!selected || !cars.has(selected as string)) {
                selected = (that.$carsDropdown.val() as string[])[0];
                that.showEntrantSkin(selected, "random_skin", $val);
            }

            $val.empty();

            for (let val of cars.values()) {
                let $opt = $("<option />");
                $opt.attr({'value': val});
                $opt.text(prettifyName(val, true));

                if (val === selected) {
                    $opt.attr({"selected": "selected"});
                }

                $val.append($opt);
            }

            EntryList.populateEntryListSkins($val, selected);
        });
    }

    private autoCompleteDrivers() {
        let opts = {
            source: this.driverNames,
            select: function (event, ui) {
                // find item.value in our entrants list
                let $row = $(event.target).closest(".entrant");

                for (let entrant of possibleEntrants) {
                    if (entrant.Name === ui.item.value) {
                        // populate
                        let $team = $row.find("input[name='EntryList.Team']");
                        let $guid = $row.find("input[name='EntryList.GUID']");

                        $team.val(entrant.Team);
                        $guid.val(entrant.GUID);

                        break;
                    }
                }
            }
        };

        $(document).on('keydown.autocomplete', ".entryListName", (e: JQuery.TriggeredEvent) => {
            $(e.currentTarget).autocomplete(opts);
        });
    }
}

declare global {
    interface JQuery {
        bootstrapSwitch: any;
    }
}