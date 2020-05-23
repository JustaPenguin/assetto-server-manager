import {prettifyName} from "./utils";

declare var possibleEntrants, availableCars, fixedSetups: any;

let $entrantTemplate: JQuery<HTMLElement> | null = null;

export class EntryList {
    private $parent: JQuery<HTMLElement>;
    private $carsDropdown: JQuery<HTMLSelectElement>;
    private driverNames: Array<string> = [];
    private $entrantsDiv: JQuery<HTMLElement> | null = null;

    public constructor($parent: JQuery<HTMLElement>, $carsDropdown: JQuery<HTMLSelectElement>) {
        this.$parent = $parent;
        this.$carsDropdown = $carsDropdown;
        this.driverNames = [];

        this.$entrantsDiv = this.$parent.find("#entrants");

        if (!this.$entrantsDiv.length) {
            return;
        }

        this.autoCompleteDrivers();

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

        $("#MaxClients").on("keydown keyup", function (e) {
            let max = parseInt($(this).attr("max")!);
            let val = parseInt($(this).val() as string);

            if (val > max
                && e.keyCode !== 46 // keycode for delete
                && e.keyCode !== 8 // keycode for backspace
            ) {
                e.preventDefault();
                $(this).val(max);
            }
        });

        this.$parent.find(".entryListCar").change((e: JQuery.ChangeEvent) => {
            this.onEntryListCarChange(e);
        });

        // When the skin is changed on all initially loaded cars
        this.$parent.find(".entryListSkin").change((e: JQuery.ChangeEvent) => {
            this.showEntrantSkin($(e.currentTarget).closest(".entrant").find(".entryListCar").val(), $(e.currentTarget).val(), $(e.currentTarget))
        });


        let that = this;

        this.populateEntryListCars();
        that.$carsDropdown.change(() => {
            this.populateEntryListCars()
        });

        that.$parent.find(".btn-delete-entrant").click((e: JQuery.ClickEvent) => {
            this.deleteEntrant(e)
        });

        that.$parent.find(".addEntrant").click((e: JQuery.ClickEvent) => {
            this.addEntrant(e);
        });
    }

    addEntrant(e: JQuery.ClickEvent): void {
        e.preventDefault();

        let $numEntrantsField = $(e.currentTarget).parent().find(".numEntrantsToAdd");
        let numEntrantsToAdd = 1;

        if ($numEntrantsField.length > 0) {
            numEntrantsToAdd = parseInt($numEntrantsField.val() as string);
        }

        let maxClients = $("#MaxClients").val()!;

        let $clonedTemplate = $entrantTemplate!.clone();
        let $lastElement = this.$parent.find(".entrant").last();

        let chosenCar = $lastElement.find("[name='EntryList.Car']").val() as string;
        let chosenSkin = $lastElement.find("[name='EntryList.Skin']").val() as string;
        let ballast = $lastElement.find("[name='EntryList.Ballast']").val() as string;
        let fixedSetup = $lastElement.find("[name='EntryList.FixedSetup']").val() as string;
        let restrictor = $lastElement.find("[name='EntryList.Restrictor']").val() as string;

        for (let i = 0; i < numEntrantsToAdd; i++) {
            if ($(".entrant:visible").length >= maxClients) {
                continue;
            }

            let $elem = $clonedTemplate.clone();
            $elem.find("input[type='checkbox']").bootstrapSwitch();
            $elem.appendTo(this.$parent.find(".entrants-block"));
            $elem.find(".entryListCar").change(this.onEntryListCarChange.bind(this));
            $elem.find(".btn-delete-entrant").click(this.deleteEntrant.bind(this));

            // when the skin changes on an added entrant
            $elem.find(".entryListSkin").change( (e: JQuery.ChangeEvent) => {
                this.showEntrantSkin($elem.find(".entryListCar").val(), $(e.currentTarget).val(), $(e.currentTarget))
            });

            if (chosenCar) {
                // dropdowns nead full <option> elements appending to them for populateEntryListCars to function correctly.
                $elem.find("[name='EntryList.Car']").append($("<option>", {
                    value: chosenCar,
                    text: prettifyName(chosenCar, true),
                    selected: true
                }));
            }

            if (chosenSkin) {
                $elem.find("[name='EntryList.Skin']").append($("<option>", {
                    value: chosenSkin,
                    text: prettifyName(chosenSkin, true),
                    selected: true
                }));
            }

            if (fixedSetup) {
                $elem.find("[name='EntryList.FixedSetup']").append($("<option>", {
                    value: fixedSetup,
                    text: fixedSetup,
                    selected: true
                }));
            }

            $elem.find("[name='EntryList.Ballast']").val(ballast);
            $elem.find("[name='EntryList.Restrictor']").val(restrictor);
            let $entrantID = $elem.find("[name='EntryList.EntrantID']");

            $entrantID.val($(".entrant:visible").length - 1);

            this.populateEntryListCars();
            this.showEntrantSkin(chosenCar, chosenSkin, $elem);

            $elem.css("display", "block");
        }

        let $savedNumEntrants = this.$parent.find(".totalNumEntrants");
        $savedNumEntrants.val(this.$parent.find(".entrant:visible").length);

        this.toggleAlreadyAutocompletedDrivers();
    }

    populateEntryListCars() {
        // populate list of cars in entry list
        let cars = new Set(this.$carsDropdown.val() as string[]);

        let that = this;

        this.$parent.find(".entryListCar").each(function (index, val) {
            let $val = $(val);
            let selected = $val.find("option:selected").val() as string;
            let $anyCar = $("<option value='any_car_model'>Any Available Car</option>");

            if (!selected || !cars.has(selected)) {
                if (selected === 'any_car_model' || cars.size < 1) {
                    selected = $anyCar.val() as string;
                } else {
                    selected = cars.values().next().value;
                }

                that.showEntrantSkin(selected, "random_skin", $val);
            }


            $val.empty();
            $anyCar.appendTo($val);

            for (let val of cars.values()) {
                let $opt = $("<option />");
                $opt.attr({'value': val});
                // use the text from the cars dropdown to populate the name, fallback to prettify if necessary
                let realCarName = that.$carsDropdown.find("option[value='" + val + "']").text();

                if (!realCarName) {
                    realCarName = prettifyName(val, true);
                }

                $opt.text(realCarName);

                if (val === selected) {
                    $opt.attr({"selected": "selected"});
                }

                $val.append($opt);
            }

            that.populateEntryListSkinsAndSetups($val, selected);
        });
    }

    showEntrantSkin(currentCar, skin, $this) {
        let fallBackImage = "/static/img/no-preview-car.png";
        let $preview = $this.closest(".entrant").find(".entryListCarPreview");

        if (currentCar === "any_car_model") {
            $preview.attr({"src": fallBackImage, "alt": "Preview Image"});
            return;
        }

        if (currentCar in availableCars && availableCars[currentCar] != null && availableCars[currentCar].length > 0) {
            if (skin === "random_skin") {
                skin = availableCars[currentCar][0]
            }

            let path = "/content/cars/" + currentCar + "/skins/" + skin + "/preview.jpg";

            $.get(path)
                .done(function () {
                    // preview for skin exists
                    $preview.attr({"src": path, "alt": prettifyName(skin, false)})
                }).fail(function () {
                // preview doesn't exist, load default fall back image
                path = fallBackImage;
                $preview.attr({"src": path, "alt": "Preview Image"})
            });
        }
    }

    onEntryListCarChange(e: JQuery.ChangeEvent) {
        let $this = $(e.currentTarget);
        let val = $this.val();

        this.populateEntryListSkinsAndSetups($this, val);

        // When the car is changed for an added entrant
        this.showEntrantSkin(val, $this.closest(".entrant").find(".entryListSkin").val(), $this)
    }

    populateEntryListSkinsAndSetups($elem, car) {
        // populate skins
        let $skinsDropdown = $elem.closest(".entrant").find(".entryListSkin");
        let selectedSkin = $skinsDropdown.val();

        $skinsDropdown.empty();

        $("<option value='random_skin'>&lt;random skin&gt;</option>").appendTo($skinsDropdown);

        try {

            if (car in availableCars && availableCars[car] != null) {
                for (let skin of availableCars[car]) {
                    let $opt = $("<option/>");
                    $opt.attr({'value': skin});
                    $opt.text(prettifyName(skin, true));

                    if (skin === selectedSkin) {
                        $opt.attr({'selected': 'selected'});
                    }

                    $opt.appendTo($skinsDropdown);
                }
            }
        } catch (e) {
            console.error(e);
        }

        // populate fixed setups
        let $fixedSetupDropdown = $elem.closest(".entrant").find(".fixedSetup");
        let selectedSetup = $fixedSetupDropdown.val();

        $fixedSetupDropdown.empty();

        $("<option>").val("").text("No Fixed Setup").appendTo($fixedSetupDropdown);

        try {
            if (car in fixedSetups && fixedSetups[car] !== null) {
                for (let track in fixedSetups[car]) {
                    // create an optgroup for the track
                    let $optgroup = $("<optgroup>").attr("label", prettifyName(track, false));

                    for (let setup of fixedSetups[car][track]) {
                        let setupFullPath = car + "/" + track + "/" + setup;

                        let $opt = $("<option/>");
                        $opt.attr({'value': setupFullPath});
                        $opt.text(prettifyName(setup.replace(".ini", ""), true));

                        if (setupFullPath === selectedSetup) {
                            $opt.attr({'selected': 'selected'});
                        }

                        $opt.appendTo($optgroup);
                    }

                    $optgroup.appendTo($fixedSetupDropdown);
                }
            }
        } catch (e) {
            console.error(e);
        }
    }

    private deleteEntrant(e: JQuery.ClickEvent) {
        e.preventDefault();

        let $raceSetup = $(e.currentTarget).closest(".race-setup");

        $(e.currentTarget).closest(".entrant").remove();

        let $savedNumEntrants = $raceSetup.find(".totalNumEntrants");
        $savedNumEntrants.val($raceSetup.find(".entrant:visible").length);

        this.toggleAlreadyAutocompletedDrivers();
    }

    private toggleAlreadyAutocompletedDrivers() {
        $(".entrant-autofill option").each(function (index, elem) {
            let found = false;
            let $elem = $(elem);

            $(".entrant .entryListName").each(function (entryIndex, entryName) {
                if ($(entryName).val() === $elem.val()) {
                    found = true;
                }
            });


            if (found) {
                $elem.hide();
            } else {
                $elem.show();
            }
        });
    }

    private autoCompleteDrivers() {
        if (!possibleEntrants) {
            return;
        }

        let that = this;

        function autoFillEntrant(elem, val) {
            let $row = $(elem).closest(".entrant");

            for (let entrant of possibleEntrants) {
                if (entrant.Name === val) {
                    // populate
                    let $team = $row.find("input[name='EntryList.Team']");
                    let $guid = $row.find("input[name='EntryList.GUID']");
                    let $name = $row.find("input[name='EntryList.Name']");

                    $name.val(entrant.Name);
                    $team.val(entrant.Team);
                    $guid.val(entrant.GUID);

                    break;
                }
            }

            that.toggleAlreadyAutocompletedDrivers();
        }

        for (let entrant of possibleEntrants) {
            $(".entrant-autofill").append($("<option>").val(entrant.Name).text(entrant.Name));
        }

        $(document).on("change", ".entrant-autofill", function (e) {
            autoFillEntrant(e.currentTarget, $(e.currentTarget).val());
        });
    }
}