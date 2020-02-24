"use strict";

import {CarSearch} from "../CarSearch";

let $document;

let moment = require("moment");

// entry-point
$(document).ready(function () {
    console.log("initialising server manager javascript");

    $document = $(document);

    // init bootstrap-switch
    $.fn.bootstrapSwitch.defaults.size = 'small';
    $.fn.bootstrapSwitch.defaults.animate = false;
    $.fn.bootstrapSwitch.defaults.onColor = "success";
    $document.find("input[type='checkbox']:not(input[name='EntryList.OverwriteAllEvents']:hidden):not(input[name='session-start-after-parent']:hidden)").bootstrapSwitch();

    championships.init();
    $document.find(".race-setup").each(function (index, elem) {
        new RaceSetup($(elem));
    });

    $document.find("#open-in-simres").each(function (index, elem) {
        let query = window.location.search;
        let link = window.location.href.replace(query, "").split("#")[0].replace("results", "results/download") + ".json" + query;

        $(elem).attr('href', "http://simresults.net/remote?result=" + link);

        return false
    });

    $document.find("#simres-group").each(function (index, elem) {
        let query = window.location.search;
        let link = window.location.href.replace(query, "") + "/export-results" + query;

        $(elem).attr('href', "http://simresults.net/remote?results=" + link);

        return false
    });

    serverLogs.init();
    initUploaders();

    $document.find('[data-toggle="tooltip"]').tooltip();

    $("[data-toggle=popover]").each(function (i, obj) {
        $(this).popover({
            html: true,
            sanitize: false,
            content: function () {
                let id = $(this).attr('id');

                return $('#popover-content-' + id).html();
            },
        });
    });

    $(".time-local").each(function (i, elem) {
        let $elem = $(elem);

        $elem.text(moment.parseZone($elem.attr("data-time")).tz(moment.tz.guess()).format('LLLL (z)'));
    });

    $(".time-local-kitchen").each(function (i, elem) {
        let $elem = $(elem);

        $elem.text(moment.parseZone($elem.attr("data-time")).tz(moment.tz.guess()).calendar());
    });

    const timezone = Intl.DateTimeFormat().resolvedOptions().timeZone;

    $(".timezone").text(timezone);
    $(".event-schedule-timezone").val(timezone);
    $(".session-schedule-timezone").val(timezone);
    $(".sol-timezone").val(timezone);

    $document.find(".row-link").click(function () {
        window.location = $(this).data("href");
    });

    $document.find(".results .driver-link").click(function () {
        window.location = $(this).data("href");
        window.scrollBy(0, -100);
    });

    $('form').submit(function () {
        $(this).find('input[type="checkbox"]').each(function () {
            let $checkbox = $(this);
            if ($checkbox.is(':checked')) {
                $checkbox.attr('value', '1');
            } else {
                $checkbox.after().append($checkbox.clone().attr({type: 'hidden', value: 0}));
                $checkbox.prop('disabled', true);
            }
        })
    });

    if ($document.find("form[data-safe-submit]").length > 0) {
        let canSubmit = false;

        $document.find("button[type='submit']").click(function () {
            canSubmit = true;
        });

        // ask the user before they close the webpage
        window.onbeforeunload = function () {
            if (canSubmit) {
                return;
            }

            return "Are you sure you want to navigate away? You'll lose unsaved changes to this setup if you do.";
        };
    }

    // Fix for mobile safari
    $document.find(".Cars").change(function () {
        let selectedCars = $(this).children("option:selected").val();

        if (selectedCars !== undefined) {
            $(this).removeAttr('required')
        } else {
            $(this).attr('required', 'required')
        }
    });

    $('#SetupFile').on('change', function () {
        let fileName = this.files[0].name;
        $(this).next('.custom-file-label').html(fileName);
    });

    $document.find("#CustomRaceScheduled").change(function () {
        if ($(this).val() && $document.find("#CustomRaceScheduledTime").val()) {
            $document.find("#start-race-button").hide();
            $document.find("#save-race-button").val("schedule");
        } else {
            $document.find("#start-race-button").show();
            $document.find("#save-race-button").val("justSave");
        }
    });

    $document.find("#CustomRaceScheduledTime").change(function () {
        if ($(this).val() && $document.find("#CustomRaceScheduled").val()) {
            $document.find("#start-race-button").hide();
            $document.find("#save-race-button").val("schedule");
        } else {
            $document.find("#start-race-button").show();
            $document.find("#save-race-button").val("justSave");
        }
    });
});

const nameRegex = /^[A-Za-z]{0,5}[0-9]+/;

function prettifyName(name, acronyms) {
    try {
        if (!name || name.length === 0) {
            return "";
        }

        let parts = name.split("_");

        if (parts[0] === "ks") {
            parts.shift();
        }

        for (let i = 0; i < parts.length; i++) {
            if ((acronyms && parts[i].length <= 3) || (acronyms && parts[i].match(nameRegex))) {
                parts[i] = parts[i].toUpperCase();
            } else {
                let split = parts[i].split(' ');

                parts[i] = split.map(w => w.length > 0 ? w[0].toUpperCase() + w.substr(1).toLowerCase() : "").join(' ');
            }
        }

        return parts.join(" ")
    } catch(error) {
         return name
    }
}


export function initMultiSelect($element) {
    $element.each(function (i, elem) {
        let $elem = $(elem);

        if ($elem.is(":hidden")) {
            return true;
        }

        $elem.multiSelect();
    });
}

let $entrantTemplate = null;

function makeCarString(cars) {
    let out = "";

    for (let index = 0; index < cars.length; index++) {
        if (index === 0) {
            out = " - " + prettifyName(cars[index], true)
        } else {
            out = out + ", " + prettifyName(cars[index], true)
        }
    }

    return out
}

class RaceSetup {
    // jQuery elements
    $trackDropdown;
    $trackLayoutDropdown;
    $trackLayoutDropdownParent;
    $carsDropdown;
    $tyresDropdown;
    $addWeatherButton;

    // the current layout as specified by the server
    currentLayout;

    // all available track layout options
    trackLayoutOpts;
    $parent;

    constructor($parent) {
        this.$parent = $parent;
        this.trackLayoutOpts = {};

        this.$carsDropdown = $parent.find(".Cars");

        this.$trackDropdown = $parent.find("#Track");
        this.$trackLayoutDropdown = $parent.find("#TrackLayout");
        this.$trackLayoutDropdownParent = this.$trackLayoutDropdown.closest(".form-group");

        this.$addWeatherButton = $parent.find("#addWeather");

        if (this.$carsDropdown) {
            initMultiSelect(this.$carsDropdown);
            this.$tyresDropdown = $parent.find("#LegalTyres");

            if (this.$tyresDropdown) {
                this.$tyresDropdown.multiSelect();

                this.$carsDropdown.change(this.populateTyreDropdown.bind(this));

                this.populateTyreDropdown();
            }
        }

        this.$addWeatherButton.click(this.addWeather.bind(this));

        $parent.find(".weather-delete").click(function (e) {
            e.preventDefault();
            let $this = $(this);

            $this.closest(".weather").remove();

            // go through all .weather divs and update their numbers
            $parent.find(".weather").each(function (index, elem) {
                $(elem).find(".weather-num").text(index);
            });
        });

        $parent.find(".weather-graphics").change(this.updateWeatherGraphics);

        let that = this;

        // restrict loading track layouts to pages which have track dropdown and layout dropdown on them.
        if (this.$trackDropdown.length && this.$trackLayoutDropdown.length) {
            // build a map of track => available layouts
            this.$trackLayoutDropdown.find("option").each(function (index, opt) {
                let $optValSplit = $(opt).val().split(":");
                let trackName = $optValSplit[0];
                let trackLayout = $optValSplit[1];

                if (!that.trackLayoutOpts[trackName]) {
                    that.trackLayoutOpts[trackName] = [];
                }

                that.trackLayoutOpts[trackName].push(trackLayout);

                if ($optValSplit.length > 2) {
                    that.currentLayout = trackLayout;
                }
            });

            that.$trackLayoutDropdownParent.hide();
            that.loadTrackLayouts();

            that.$trackDropdown.change(that.loadTrackLayouts.bind(this));
            that.$trackLayoutDropdown.change(that.showTrackDetails.bind(this));

        }

        this.raceLaps();
        this.showEnabledSessions();
        this.showSolSettings();
        this.toggleOverridePassword();

        this.initEntrantsList();
        this.initSunAngle();
        this.initSurfacePresets();

        this.initPickupModeWatcher();
    }

    initPickupModeWatcher() {
        let that = this;

        let $pickupModeSwitch = $("#PickupModeEnabled");
        let $lockedEntryListSwitch = $("#LockedEntryList");
        let $reversedGridRacePositions = $("#ReversedGridRacePositions");

        this.setPickupSwitches();

        $pickupModeSwitch.on('switchChange.bootstrapSwitch', function (event, state) {
            that.setPickupSwitches();
        });

        $lockedEntryListSwitch.on('switchChange.bootstrapSwitch', function (event, state) {
            that.setPickupSwitches();
        });

        $reversedGridRacePositions.on('change', function () {
            that.setPickupSwitches();
        });
    }

    setPickupSwitches() {
        let $lockedEntryListSwitch = $("#LockedEntryList");
        let $lockedReverseWarning = $("#locked-reverse-warning");

        let pickup = $("#PickupModeEnabled").bootstrapSwitch('state');
        let reversed = $("#ReversedGridRacePositions").val();

        if (pickup === false && reversed != 0) {
            $lockedEntryListSwitch.bootstrapSwitch('state', false);
            $lockedEntryListSwitch.bootstrapSwitch('disabled', true);

            $lockedReverseWarning.show();
        } else {
            $lockedEntryListSwitch.bootstrapSwitch('disabled', false);

            $lockedReverseWarning.hide();
        }
    }

    updateWeatherGraphics() {
        let $this = $(this);

        $this.closest(".weather").find(".weather-preview").attr({
            'src': '/content/weather/' + $this.val() + '/preview.jpg',
            'alt': $this.val(),
        });
    }

    /**
     * add weather elements to the form when the 'new weather' button is clicked
     */
    addWeather(e) {
        e.preventDefault();

        let $oldWeather = this.$parent.find(".weather").last();

        let $newWeather = $oldWeather.clone(true, true);
        $newWeather.find(".weather-num").text(this.$parent.find(".weather").length);
        $newWeather.find(".weather-delete").show();

        $oldWeather.after($newWeather);
    }

    /**
     * when a session 'enabled' checkbox is modified, toggle the state of the session-details element
     */
    showEnabledSessions() {
        let that = this;


        let $hiddenWhenBookingEnabled = that.$parent.find(".hidden-booking-enabled");
        let $visibleWhenBookingEnabled = that.$parent.find(".visible-booking-enabled");

        if ($(".session-enabler[name='Booking.Enabled']").is(":checked")) {
            $hiddenWhenBookingEnabled.hide();
            $visibleWhenBookingEnabled.show();
        } else {
            $hiddenWhenBookingEnabled.show();
            $visibleWhenBookingEnabled.hide();
        }

        if (!$("#race-weekend-session").length) {
            // session enabling is very different in race weekends, don't interfere with that here.
            $(".session-enabler").each(function (index, elem) {
                $(elem).on('switchChange.bootstrapSwitch', function (event, state) {
                    let $this = $(this);
                    let $elem = $this.closest(".tab-pane").find(".session-details");
                    let $panelLabel = that.$parent.find("#" + $this.closest(".tab-pane").attr("aria-labelledby"));

                    let isBooking = $(elem).attr("name") === "Booking.Enabled";

                    if (state) {
                        $elem.show();
                        $panelLabel.addClass("text-success");

                        if (isBooking) {
                            $hiddenWhenBookingEnabled.hide();
                            $visibleWhenBookingEnabled.show();
                        }
                    } else {
                        $elem.hide();
                        $panelLabel.removeClass("text-success");

                        if (isBooking) {
                            $hiddenWhenBookingEnabled.show();
                            $visibleWhenBookingEnabled.hide();
                        }
                    }
                });
            });
        }
    }

    /**
     * show the override password text field when the checkbox is modified
     */
    toggleOverridePassword() {
        $("#OverridePassword").on('switchChange.bootstrapSwitch', function (event, state) {
            let $this = $(this);
            let $replacementPasswordElem = $this.closest(".card-body").find("#ReplacementPasswordWrapper");

            if (state) {
                $replacementPasswordElem.show();
            } else {
                $replacementPasswordElem.hide();
            }
        });
    }

    /**
     * when a Sol 'enabled' checkbox is modified, toggle the state of the sol-settings and not-sol-settings elements
     */
    showSolSettings() {
        let that = this;

        $(".sol-enabler").each(function (index, elem) {
            that.showSolWeathers($(elem).is(':checked'));

            $(elem).on('switchChange.bootstrapSwitch', function (event, state) {
                let $this = $(this);
                let $solElem = $this.closest(".card-body").find(".sol-settings");
                let $notSolElem = $this.closest(".card-body").find(".not-sol-settings");

                if (state) {
                    $solElem.show();
                    $notSolElem.hide();
                } else {
                    $solElem.hide();
                    $notSolElem.show();
                }

                that.showSolWeathers(state);
            });
        });
    }

    /**
     * hide non-sol weather if sol is enabled.
     *
     * @param state
     */
    showSolWeathers(state) {
        this.$parent.find(".weather-graphics").each(function (graphicsIndex, graphicsElement) {
            let $elem = $(graphicsElement);
            let $opts = $elem.find("option");
            let $selectedOpt = $elem.find("option:selected");

            if (state) {
                if (!/sol/i.test($selectedOpt.val())) {
                    $elem.val("sol_01_CLear");
                }
            }

            for (let i = 0; i < $opts.length; i++) {
                let $opt = $($opts[i]);

                if (state && !/sol/i.test($opt.val())) {
                    $opt.hide();
                } else {
                    $opt.show();
                }
            }
        });
    }


    /**
     * populate the tyre dropdown for all currently selected cars.
     */
    populateTyreDropdown() {
        // quick race doesn't have tyre set up.
        if (typeof availableTyres === "undefined" || !this.$carsDropdown.length) {
            return
        }

        let cars = this.$carsDropdown.val();

        let allValidTyres = new Set();
        let tyreCars = {};

        for (let index = 0; index < cars.length; index++) {
            let car = cars[index];
            let carTyres = availableTyres[car];

            for (let tyre in carTyres) {
                if (!tyreCars[tyre]) {
                    tyreCars[tyre] = []
                }

                tyreCars[tyre].push(car)
            }
        }

        for (let index = 0; index < cars.length; index++) {
            let car = cars[index];
            let carTyres = availableTyres[car];

            for (let tyre in carTyres) {
                allValidTyres.add(tyre);

                let $dropdownTyre = this.$tyresDropdown.find("option[value='" + tyre + "']");

                if ($dropdownTyre.length) {
                    $dropdownTyre.text(carTyres[tyre] + " (" + tyre + ")" + makeCarString(tyreCars[tyre]));
                    this.$tyresDropdown.multiSelect('refresh');
                    continue; // this has already been added
                }

                this.$tyresDropdown.multiSelect('addOption', {
                    'value': tyre,
                    'text': carTyres[tyre] + " (" + tyre + ")" + makeCarString(tyreCars[tyre]),
                });

                this.$tyresDropdown.multiSelect('select', tyre);
            }
        }

        let that = this;

        this.$tyresDropdown.find("option").each(function (index, elem) {
            let $elem = $(elem);

            if (!allValidTyres.has($elem.val())) {
                $elem.remove();

                that.$tyresDropdown.multiSelect('refresh');
            }
        });
    }

    /**
     * given a dropdown input which specifies 'laps'/'time', raceLaps will show the correct input element
     * and empty the unneeded one for either laps or race time.
     */
    raceLaps() {
        let $timeOrLaps = this.$parent.find("#TimeOrLaps");
        let $raceLaps = this.$parent.find("#RaceLaps");
        let $raceTime = this.$parent.find("#RaceTime");
        let $extraLap = this.$parent.find(".race-extra-lap");

        if ($timeOrLaps.length) {
            $timeOrLaps.change(function () {
                let selected = $timeOrLaps.find("option:selected").val();

                if (selected === "Time") {
                    $raceLaps.find("input").val(0);
                    $raceTime.find("input").val(15);
                    $raceLaps.hide();
                    $raceTime.show();

                    if ($extraLap.length > 0) {
                        $extraLap.show();
                    }
                } else {
                    $raceTime.find("input").val(0);
                    $raceLaps.find("input").val(10);
                    $raceLaps.show();
                    $raceTime.hide();

                    if ($extraLap.length > 0) {
                        $extraLap.hide();
                    }
                }
            });
        }
    }

    /**
     * show track details shows the correct image and pit boxes for the track/layout combo
     */
    showTrackDetails() {
        let track = this.$trackDropdown.val();
        let layout = this.$trackLayoutDropdown.val();

        let src = '/content/tracks/' + track + '/ui';

        if (layout && layout !== '<default>') {
            src += '/' + layout;
        }

        let jsonURL = src + "/ui_track.json";

        src += '/preview.png';

        this.$parent.find("#trackImage").attr({
            'src': src,
            'alt': track + ' ' + layout,
        });

        let $pitBoxes = $document.find("#track-pitboxes");
        let $maxClients = $document.find("#MaxClients");
        let $pitBoxesWarning = $document.find("#track-pitboxes-warning");

        $.getJSON(jsonURL, function (trackInfo) {
            $pitBoxes.closest(".row").show();
            $pitBoxes.text(trackInfo.pitboxes);

            let $entrantIDs = $document.find(".entrant-id");
            $entrantIDs.attr("max", (trackInfo.pitboxes - 1));

            let entrants = $document.find(".entrant").length;

            if (entrants > trackInfo.pitboxes) {
                $pitBoxesWarning.show()
            } else {
                $pitBoxesWarning.hide()
            }

            let overrideAmount = $maxClients.data('value-override');

            if ((overrideAmount && trackInfo.pitboxes <= overrideAmount) || !overrideAmount) {
                $maxClients.attr("max", trackInfo.pitboxes);

                if (parseInt($maxClients.val()) > trackInfo.pitboxes) {
                    $maxClients.val(trackInfo.pitboxes);
                }
            } else if (overrideAmount) {
                $maxClients.attr("max", overrideAmount);
                $maxClients.val(overrideAmount);
            }
        })
            .fail(function () {
                $pitBoxes.closest(".row").hide()
            })
    }

    /**
     * loadTrackLayouts: looks at the selected track and loads in the correct layouts for it into the
     * track layout dropdown
     */
    loadTrackLayouts() {
        this.$trackLayoutDropdown.empty();

        let selectedTrack = this.$trackDropdown.find("option:selected").val();
        let availableLayouts = this.trackLayoutOpts[selectedTrack];

        if (availableLayouts && !(availableLayouts.length === 1 && availableLayouts[0] === "<default>")) {
            for (let i = 0; i < availableLayouts.length; i++) {
                this.$trackLayoutDropdown.append(this.buildTrackLayoutOption(availableLayouts[i]));
            }

            this.$trackLayoutDropdownParent.show();
        } else {
            // add an option with an empty value
            this.$trackLayoutDropdown.append(this.buildTrackLayoutOption(""));
            this.$trackLayoutDropdownParent.hide();
        }


        this.showTrackDetails();
    }

    /**
     * buildTrackLayoutOption: builds an <option> containing track layout information
     * @param layout
     * @returns {HTMLElement}
     */
    buildTrackLayoutOption(layout) {
        let $opt = $("<option/>");
        $opt.attr({'value': layout});
        $opt.text(prettifyName(layout, true));

        if (layout === this.currentLayout) {
            $opt.prop("selected", true);
        }

        return $opt;
    }

    $entrantsDiv;

    driverNames;

    toggleAlreadyAutocompletedDrivers() {
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

    autoCompleteDrivers() {
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

        let opts = {
            source: this.driverNames,
            select: function (event, ui) {
                autoFillEntrant(event.target, ui.item.value);
            }
        };


        for (let entrant of possibleEntrants) {
            $(".entrant-autofill").append($("<option>").val(entrant.Name).text(entrant.Name));
        }

        $(document).on("change", ".entrant-autofill", function (e) {
            autoFillEntrant(e.currentTarget, $(e.currentTarget).val());
        });
    }

    initEntrantsList() {
        this.driverNames = [];

        this.$entrantsDiv = this.$parent.find("#entrants");

        if (!this.$entrantsDiv.length) {
            return;
        }

        if (possibleEntrants) {
            for (let entrant of possibleEntrants) {
                this.driverNames.push(entrant.Name);
            }
        }

        function onEntryListCarChange() {
            let $this = $(this);
            let val = $this.val();

            populateEntryListSkinsAndSetups($this, val);

            // When the car is changed for an added entrant
            showEntrantSkin(val, $this.closest(".entrant").find(".entryListSkin").val(), $this)
        }

        function showEntrantSkin(currentCar, skin, $this) {
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

                let path = "/content/cars/" + encodeURIComponent(currentCar) + "/skins/" + encodeURIComponent(skin) + "/preview.jpg";

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

        $("#MaxClients").on("keydown keyup", function (e) {
            let max = parseInt($(this).attr("max"));
            let val = parseInt($(this).val());

            if (val > max
                && e.keyCode !== 46 // keycode for delete
                && e.keyCode !== 8 // keycode for backspace
            ) {
                e.preventDefault();
                $(this).val(max);
            }
        });

        this.$parent.find(".entryListCar").change(onEntryListCarChange);

        // When the skin is changed on all initially loaded cars
        this.$parent.find(".entryListSkin").change(function () {
            showEntrantSkin($(this).closest(".entrant").find(".entryListCar").val(), $(this).val(), $(this))
        });
        this.autoCompleteDrivers();

        // initialise entrantTemplate if it's null. this will only happen once so cloned race setups
        // have an entrant template to work from.
        let $tmpl = this.$parent.find("#entrantTemplate");

        if (!$entrantTemplate) {
            $entrantTemplate = $tmpl.prop("id", "").clone(true, true);
        }

        $tmpl.remove();

        let that = this;

        function populateEntryListSkinsAndSetups($elem, car) {
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

        function deleteEntrant(e) {
            e.preventDefault();

            let $raceSetup = $(this).closest(".race-setup");

            $(this).closest(".entrant").remove();


            let $savedNumEntrants = $raceSetup.find(".totalNumEntrants");
            $savedNumEntrants.val($raceSetup.find(".entrant:visible").length);


            that.toggleAlreadyAutocompletedDrivers();
        }

        function populateEntryListCars() {
            // populate list of cars in entry list
            let cars = new Set(that.$carsDropdown.val());

            that.$parent.find(".entryListCar").each(function (index, val) {
                let $val = $(val);
                let selected = $val.find("option:selected").val();
                let $anyCar = $("<option value='any_car_model'>Any Available Car</option>");

                if (!selected || !cars.has(selected)) {
                    if (selected === 'any_car_model' || cars.size < 1) {
                        selected = $anyCar.val();
                    } else {
                        selected = cars.values().next().value;
                    }

                    showEntrantSkin(selected, "random_skin", $val);
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

                populateEntryListSkinsAndSetups($val, selected);
            });
        }

        populateEntryListCars();
        that.$carsDropdown.change(populateEntryListCars);
        that.$parent.find(".btn-delete-entrant").click(deleteEntrant);

        that.$parent.find(".addEntrant").click(function (e) {
            e.preventDefault();

            let $numEntrantsField = $(this).parent().find(".numEntrantsToAdd");
            let numEntrantsToAdd = 1;

            if ($numEntrantsField.length > 0) {
                numEntrantsToAdd = $numEntrantsField.val();
            }

            let maxClients = $("#MaxClients").val();

            let $clonedTemplate = $entrantTemplate.clone();
            let $lastElement = that.$parent.find(".entrant").last();

            let chosenCar = $lastElement.find("[name='EntryList.Car']").val();
            let chosenSkin = $lastElement.find("[name='EntryList.Skin']").val();
            let ballast = $lastElement.find("[name='EntryList.Ballast']").val();
            let fixedSetup = $lastElement.find("[name='EntryList.FixedSetup']").val();
            let restrictor = $lastElement.find("[name='EntryList.Restrictor']").val();

            for (let i = 0; i < numEntrantsToAdd; i++) {
                if ($(".entrant:visible").length >= maxClients) {
                    continue;
                }

                let $elem = $clonedTemplate.clone();
                $elem.find("input[type='checkbox']").bootstrapSwitch();
                $elem.appendTo(that.$parent.find(".entrants-block"));
                $elem.find(".entryListCar").change(onEntryListCarChange);
                $elem.find(".btn-delete-entrant").click(deleteEntrant);

                // when the skin changes on an added entrant
                $elem.find(".entryListSkin").change(function () {
                    showEntrantSkin($elem.find(".entryListCar").val(), $(this).val(), $(this))
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

                populateEntryListCars();
                showEntrantSkin(chosenCar, chosenSkin, $elem);

                $elem.css("display", "block");
            }

            let $savedNumEntrants = that.$parent.find(".totalNumEntrants");
            $savedNumEntrants.val(that.$parent.find(".entrant:visible").length);

            that.toggleAlreadyAutocompletedDrivers();
        });
    }


    initSunAngle() {
        let $timeOfDay = this.$parent.find("#TimeOfDay");
        let $sunAngle = this.$parent.find("#SunAngle");

        function updateTime() {
            let angle = $sunAngle.val();
            let time = getTime(angle);

            $timeOfDay.val(time.getHours() + ":" + time.getFullMinutes());
        }

        updateTime();

        $timeOfDay.change(function () {
            let split = $(this).val().split(':');

            if (split.length < 2) {
                return;
            }

            $sunAngle.val(getSunAngle(split[0], split[1]));
        });

        $sunAngle.change(updateTime);
    }

    initSurfacePresets() {
        let $surfacePresetDropdown = this.$parent.find("#SurfacePreset");

        if (!$surfacePresetDropdown.length) {
            return;
        }

        let $sessionStart = this.$parent.find("#SessionStart");
        let $randomness = this.$parent.find("#Randomness");
        let $sessionTransfer = this.$parent.find("#SessionTransfer");
        let $lapGain = this.$parent.find("#LapGain");

        $surfacePresetDropdown.change(function () {
            let val = $surfacePresetDropdown.val();

            if (val === "") {
                return;
            }

            let preset = surfacePresets[val];

            $sessionStart.val(preset["SessionStart"]);
            $randomness.val(preset["Randomness"]);
            $sessionTransfer.val(preset["SessionTransfer"]);
            $lapGain.val(preset["LapGain"]);
        });
    }
}

function pad(num, size) {
    let s = num + "";
    while (s.length < size) {
        s = "0" + s;
    }
    return s;
}

function msToTime(s) {
    let hours = (s / 3.6e6 | 0);
    let minutes = ((s % 3.6e6) / 6e4 | 0);
    let seconds = ((s % 6e4) / 1000 | 0) + '.' + pad(s % 1000, 3);

    let output = '';

    if (hours !== 0) {
        output += hours + "h"
    }

    if (minutes !== 0) {
        output += minutes + "m"
    }

    if (seconds !== '0.0') {
        output += seconds + "s"
    }

    return output
}

function timeDiff(tstart, tend) {
    let diff = Math.floor((tend - tstart) / 1000), units = [
        {d: 60, l: "s"},
        {d: 60, l: "m"},
        {d: 24, l: "hr"},
    ];

    let s = '';
    for (let i = 0; i < units.length; i++) {
        if (diff === 0) {
            continue
        }
        s = (diff % units[i].d) + units[i].l + " " + s;
        diff = Math.floor(diff / units[i].d);
    }
    return s;
}

Date.prototype.getFullMinutes = function () {
    if (this.getMinutes() < 10) {
        return '0' + this.getMinutes();
    }
    return this.getMinutes();
};

// get time from sun angle: https://github.com/Pringlez/ACServerManager/blob/master/frontend/app/controllers.js
function getTime(sunAngle) {
    let baseLine = new Date(2000, 1, 1, 13, 0, 0, 0);
    let multiplier = (sunAngle / 16) * 60;
    baseLine.setMinutes(baseLine.getMinutes() + multiplier);

    return baseLine;
}

// get sun angle from time: https://github.com/Pringlez/ACServerManager/blob/master/frontend/app/controllers.js
function getSunAngle(hours, mins) {
    let baseLine = new Date(2000, 1, 1, 13, 0, 0, 0);
    let time = new Date(2000, 1, 1, hours, mins, 0);
    let diff = time - baseLine;

    return Math.round(((diff / 60000) / 60) * 16);
}

let serverLogs = {
    init: function () {
        let $serverLog = $document.find("#server-logs");
        let $managerLog = $document.find("#manager-logs");
        let $pluginLog = $document.find("#plugin-logs");

        let disableServerLogRefresh = false;
        let disableManagerLogRefresh = false;
        let disablePluginLogRefresh = false;

        $serverLog.on("mousedown", function () {
            disableServerLogRefresh = true;
        });

        $serverLog.on("mouseup", function () {
            disableServerLogRefresh = false;
        });

        $pluginLog.on("mousedown", function () {
            disablePluginLogRefresh = true;
        });

        $pluginLog.on("mouseup", function () {
            disablePluginLogRefresh = false;
        });

        $managerLog.on("mousedown", function () {
            disableManagerLogRefresh = true;
        });

        $managerLog.on("mouseup", function () {
            disableManagerLogRefresh = false;
        });

        function isAtBottom($elem) {
            let node = $elem[0];
            return node.scrollTop + node.offsetHeight >= node.scrollHeight - 40;
        }

        if ($serverLog.length && $managerLog.length && $pluginLog.length) {
            setInterval(function () {
                $.get("/api/logs", function (data) {
                    if (!window.getSelection().toString()) {

                        if (isAtBottom($serverLog) && !disableServerLogRefresh) {
                            $serverLog.text(data.ServerLog);
                            $serverLog.scrollTop(1E10);
                        }

                        if (isAtBottom($managerLog) && !disableManagerLogRefresh) {
                            $managerLog.text(data.ManagerLog);
                            $managerLog.scrollTop(1E10);
                        }

                        if (isAtBottom($pluginLog) && !disablePluginLogRefresh) {
                            $pluginLog.text(data.PluginsLog);
                            $pluginLog.scrollTop(1E10);
                        }
                    }
                });
            }, 1000);
        }
    },
};


function initUploaders() {
    $("#input-folder-car").on("change", function () {
        handleCarFiles(this.files);
    });

    $("#drop-zone.car-drop").on("drop", function (e) {
        handleCarDropFiles(e);
    });

    $("#drop-zone").on("dragover", dragOverHandler);
    $("#drop-zone").on("dragleave", dragOutHandler);
    $("#only-ks").on("switchChange.bootstrapSwitch", toggleKS);

    $("#input-folder-track").on("change", function () {
        handleTrackFiles(this.files);
    });

    $("#drop-zone.track-drop").on("drop", function (e) {
        handleTrackDropFiles(e);
    });

    $("#input-folder-weather").on("change", function () {
        handleWeatherFiles(this.files);
    });

    $("#drop-zone.weather-drop").on("drop", function (e) {
        handleWeatherDropFiles(e);
    });
}


function postWithProgressBar(path, data, onSuccess, onFail, $progressBar) {
    $progressBar.closest(".progress").show();
    $progressBar.removeClass("bg-success");

    function showProgress(evt) {
        if (evt.lengthComputable) {
            let percentComplete = Math.round((evt.loaded / evt.total) * 100);
            $progressBar.css('width', percentComplete + '%').attr('aria-valuenow', percentComplete);
            $progressBar.text(percentComplete + "%");

            if (percentComplete === 100) {
                $progressBar.addClass("bg-success");
            }
        }
    }

    $.ajax({
        xhr: function () {
            let xhr = new window.XMLHttpRequest();
            xhr.upload.addEventListener("progress", showProgress, false);
            xhr.addEventListener("progress", showProgress, false);
            return xhr;
        },
        type: 'POST',
        url: path,
        data: data,
        success: onSuccess,
        fail: onFail,
    });
}

const layout = {
    preview: "",
    details: "",
};

let filesToUpload = [];

function submitFiles(path) {
    //JSON encode filestoUpload, JQUERY post request to api endpoint (/api/track/car/upload)
    let newFiles = [];
    let count = 0;

    for (let x = 0; x < filesToUpload.length; x++) {
        // Encode and upload, don't post until all files are read
        let reader = new FileReader();

        reader.readAsDataURL(filesToUpload[x]);

        reader.addEventListener("load", function () {
            newFiles.push({
                'name': filesToUpload[x].name,
                'size': filesToUpload[x].size,
                'type': filesToUpload[x].type,
                'filepath': filesToUpload[x].filepath,
                'dataBase64': reader.result.toString()
            });

            count++;

            if (count === filesToUpload.length) {
                postWithProgressBar(path, JSON.stringify(newFiles), onSuccess, onFail, $("#progress-bar"));
            }
        });
    }
}

function onSuccess(data) {
    console.log("Track Successfully Added");
    location.reload(); // reload for flashes
}

function onFail(data) {
    console.log("Track Could Not be Added");
    location.reload(); // reload for flashes
}

function handleWeatherFiles(fileList) {
    // Correct filepath
    for (let x = 0; x < fileList.length; x++) {
        if (!fileList[x].filepath) {
            fileList[x].filepath = fileList[x].webkitRelativePath;
        }
    }

    let idPos = 0;

    if (fileList[0].filepath.startsWith("weather/")) {
        idPos = 2
    } else {
        idPos = 1
    }

    let splitList = {};

    for (let y = 0; y < fileList.length; y++) {
        let splitPath = fileList[y].filepath.split("/");

        let weatherIdentifier = splitPath.slice(0, idPos).join(":");

        fileList[y].newPath = splitPath.slice(1, splitPath.length - 1).join("/");

        if (!splitList[weatherIdentifier]) {
            splitList[weatherIdentifier] = []
        }

        splitList[weatherIdentifier].push(fileList[y]);
    }

    for (let weather in splitList) {
        handleWeatherFilesLoop(splitList[weather]);
    }
}

function handleWeatherFilesLoop(fileList) {
    let filesToUploadLocal = [];
    let goodFile = false;

    for (let x = 0; x < fileList.length; x++) {
        // Find the files that the server is interested in
        if (fileList[x].name === "weather.ini" || fileList[x].name.startsWith("preview.")) {
            filesToUploadLocal.push(fileList[x]);

            goodFile = true;
        }
    }

    if (!goodFile) {
        return
    }

    // Preview panel for the weather preset
    let $weatherPanel = $("#weather-info-panel");
    let $row = $("<div/>");
    let $title = $("<h3/>");
    let previewDone = false;

    let weatherName = "";

    if (fileList[0].filepath.startsWith("weather/")) {
        weatherName = fileList[0].filepath.replace('\\', '/').split("/")[1];
    } else {
        weatherName = fileList[0].filepath.replace('\\', '/').split("/")[0];
    }

    $weatherPanel.attr({'class': "card p-3 mt-2"});
    $title.text("Preview: " + weatherName);
    $row.attr({'class': "card-deck"});

    $weatherPanel.append($title);

    $weatherPanel.append($row);

    for (let x = 0; x < filesToUploadLocal.length; x++) {

        // Get a preview image, display livery name
        if (filesToUploadLocal[x].name.startsWith("preview.") && !previewDone) {
            previewDone = true;

            // Set preview to base64 encoded image
            let reader = new FileReader();

            reader.readAsDataURL(filesToUploadLocal[x]);

            reader.addEventListener("load", function () {
                $row.append(buildInfoPanel(reader.result.toString(), "Weather Preview"));
            });
        }
    }

    // Create an upload button that sends queued files to the server
    let $uploadButton = $("#upload-button");
    $uploadButton.attr({'class': "d-inline"});

    if (filesToUploadLocal.length === 0) {
        $uploadButton.text("Sorry, the files you uploaded don't seem to be a compatible weather preset!");
        $uploadButton.empty()
    } else {
        if (!$("#weather-upload-button").length) {
            let $button = $("<button/>");
            $button.attr({
                'class': "btn btn-primary",
                'id': "weather-upload-button"
            });

            $button.click(function (e) {
                e.preventDefault();
                submitFiles("/api/weather/upload")
            });
            $button.text("Upload Weather Preset(s)");

            $uploadButton.append($button);
        }

        for (let x = 0; x < filesToUploadLocal.length; x++) {
            filesToUpload.push(filesToUploadLocal[x])
        }
    }
}

let onlyKS = false;

function toggleKS() {
    onlyKS = $document.find("#only-ks").is(':checked');
}

function handleCarFiles(fileList) {
    // Correct filepath
    for (let x = 0; x < fileList.length; x++) {
        if (!fileList[x].filepath) {
            fileList[x].filepath = fileList[x].webkitRelativePath;
        }
    }

    let idPos = 0;

    if (fileList[0].filepath.startsWith("cars/")) {
        idPos = 2
    } else {
        idPos = 1
    }

    let splitList = {};

    for (let y = 0; y < fileList.length; y++) {
        // if onlyKS is set, and the file doesn't contain the ks_ prefix
        if (onlyKS && fileList[y].filepath.indexOf("ks_", 0) === -1) {
            continue
        }

        let splitPath = fileList[y].filepath.split("/");

        let carIdentifier = splitPath.slice(0, idPos).join(":");

        fileList[y].newPath = splitPath.slice(1, splitPath.length - 1).join("/");

        if (!splitList[carIdentifier]) {
            splitList[carIdentifier] = []
        }

        splitList[carIdentifier].push(fileList[y]);
    }

    for (let car in splitList) {
        handleCarFilesLoop(splitList[car]);
    }
}

function handleCarFilesLoop(fileList) {
    let filesToUploadLocal = [];
    let goodFile = false;

    for (let x = 0; x < fileList.length; x++) {
        // Find the files that the server is interested in
        if (fileList[x].name === "data.acd" || fileList[x].name === "tyres.ini" || fileList[x].name === "ui_car.json"
            || fileList[x].name.startsWith("livery.") || fileList[x].name.startsWith("preview.")
            || fileList[x].name === "ui_skin.json" || fileList[x].filepath.indexOf("/data/") !== -1) {

            filesToUploadLocal.push(fileList[x]);

            if (fileList[x].name === "ui_car.json") {
                goodFile = true;
            }
        }
    }

    if (!goodFile) {
        notA("car");
        return
    }

    let $panel = $("#car-fail");
    $panel.hide();

    // Preview panel for the car
    let $carPanel = $("#car-info-panel");
    let $row = $("<div/>");
    let $title = $("<h3/>");
    let previewDone = false;

    let entrySplit = fileList[0].filepath.replace('\\', '/').split("/");
    let carName = entrySplit[entrySplit.length - 2];

    if (fileList[0].filepath.startsWith("cars/")) {
        carName = fileList[0].filepath.split("/")[1];
    } else {
        carName = fileList[0].filepath.split("/")[0];
    }

    $carPanel.attr({'class': "card p-3 mt-2"});
    $title.text("Preview: " + carName);
    $row.attr({'class': "card-deck"});

    $carPanel.append($title);

    $carPanel.append($row);

    for (let x = 0; x < filesToUploadLocal.length; x++) {

        // Get a preview image, display livery name
        if (filesToUploadLocal[x].name.startsWith("preview.") && !previewDone) {
            previewDone = true;

            let filePathCorrected = filesToUploadLocal[x].filepath.replace('\\', '/');
            let filePathSplit = filePathCorrected.split("/");

            let skinName = filePathSplit[filePathSplit.length - 2];

            // Set preview to base64 encoded image
            let reader = new FileReader();

            reader.readAsDataURL(filesToUploadLocal[x]);

            reader.addEventListener("load", function () {
                $row.append(buildInfoPanel(reader.result.toString(), "Livery: " + skinName));
            });
        }

        // Get info about the car to display in the preview, this often fails due to bad JSON encoding
        if (filesToUploadLocal[x].name === "ui_car.json") {
            let reader = new FileReader();

            reader.readAsText(filesToUploadLocal[x]);

            reader.addEventListener("load", function () {
                let parsed = "";
                let badJSONnoDonut = false;

                try {
                    parsed = JSON.parse(reader.result.toString());
                } catch (error) {
                    badJSONnoDonut = true;
                }

                if (!badJSONnoDonut) {
                    $carPanel.append(buildHtmlTable([parsed]));
                }
            });
        }
    }

    // Create an upload button that sends queued files to the server
    let $uploadButton = $("#upload-button");
    $uploadButton.attr({'class': "d-inline"});

    if (filesToUploadLocal.length === 0) {
        $uploadButton.text("Sorry, the files you uploaded don't seem to be a compatible car!");
        $uploadButton.empty()
    } else {
        if (!$("#car-upload-button").length) {
            let $button = $("<button/>");
            $button.attr({
                'class': "btn btn-primary",
                'id': "car-upload-button"
            });

            $button.click(function (e) {
                e.preventDefault();

                submitFiles("/api/car/upload")
            });
            $button.text("Upload Car(s)");

            $uploadButton.append($button);
        }

        for (let x = 0; x < filesToUploadLocal.length; x++) {
            filesToUpload.push(filesToUploadLocal[x])
        }
    }
}

function getFilesWebkitDataTransferItems(dataTransferItems) {
    function traverseFileTreePromise(item, path = '') {
        return new Promise(resolve => {
            if (item.isFile) {
                item.file(file => {
                    file.filepath = path + file.name; //save full path
                    files.push(file);
                    resolve(file)
                })
            } else if (item.isDirectory) {
                let dirReader = item.createReader();
                dirReader.readEntries(entries => {
                    let entriesPromises = [];

                    for (let entr of entries) {
                        entriesPromises.push(traverseFileTreePromise(entr, path + item.name + "/"));
                    }

                    resolve(Promise.all(entriesPromises))
                })
            }
        })
    }

    let files = [];
    return new Promise((resolve, reject) => {
        let entriesPromises = [];

        for (let it of dataTransferItems) {
            entriesPromises.push(traverseFileTreePromise(it.webkitGetAsEntry()));
        }

        Promise.all(entriesPromises)
            .then(entries => {
                resolve(files)
            })
    })
}

function handleTrackDropFiles(ev) {
    // Prevent default behavior (Prevent file from being opened)
    ev.preventDefault();

    dragOutHandler(ev);

    let items = event.dataTransfer.items;
    getFilesWebkitDataTransferItems(items)
        .then(files => {
            handleTrackFiles(files);
        })
}

function handleCarDropFiles(ev) {
    // Prevent default behavior (Prevent file from being opened)
    ev.preventDefault();

    dragOutHandler(ev);

    let items = event.dataTransfer.items;
    getFilesWebkitDataTransferItems(items)
        .then(files => {
            handleCarFiles(files);
        })
}

function handleWeatherDropFiles(ev) {
    // Prevent default behavior (Prevent file from being opened)
    ev.preventDefault();

    dragOutHandler(ev);

    let items = event.dataTransfer.items;
    getFilesWebkitDataTransferItems(items)
        .then(files => {
            handleWeatherFiles(files);
        })
}

function dragOverHandler(ev) {
    // Prevent default behavior (Prevent file from being opened)
    ev.preventDefault();

    document.getElementById("drop-zone").classList.add('drop-zone-hovered');
}

function dragOutHandler(ev) {
    // Prevent default behavior (Prevent file from being opened)
    ev.preventDefault();

    document.getElementById("drop-zone").classList.remove('drop-zone-hovered');
}

function handleTrackFiles(fileList) {
    // Correct filepath
    for (let x = 0; x < fileList.length; x++) {
        if (!fileList[x].filepath) {
            fileList[x].filepath = fileList[x].webkitRelativePath;
        }
    }

    let idPos = 0;

    if (fileList[0].filepath.startsWith("tracks/")) {
        idPos = 2
    } else {
        idPos = 1
    }

    let splitList = {};

    for (let y = 0; y < fileList.length; y++) {
        // if onlyKS is set, and the file doesn't contain the ks_ prefix
        if (onlyKS && fileList[y].filepath.indexOf("ks_", 0) === -1) {
            continue
        }

        let splitPath = fileList[y].filepath.split("/");

        let trackIdentifier = splitPath.slice(0, idPos).join(":");

        fileList[y].newPath = splitPath.slice(1, splitPath.length - 1).join("/");

        if (!splitList[trackIdentifier]) {
            splitList[trackIdentifier] = []
        }

        splitList[trackIdentifier].push(fileList[y]);
    }

    for (let track in splitList) {
        handleTrackFilesLoop(splitList[track]);
    }

}

function handleTrackFilesLoop(fileList) {
    let layouts = {};
    let layoutNum = 0;
    let filesToUploadLocal = [];
    let trackName = "";
    let goodFile = false;

    for (let x = 0; x < fileList.length; x++) {
        // get model/surfaces and drs zones and ui folder
        if ((fileList[x].name.startsWith("models") && fileList[x].name.endsWith(".ini")) ||
            (fileList[x].name === "surfaces.ini" || fileList[x].name === "drs_zones.ini") ||
            (fileList[x].filepath.includes("/ui/") || fileList[x].name === "map.png" || fileList[x].name === "map.ini")) {

            filesToUploadLocal.push(fileList[x]);
        }

        if (fileList[x].name.startsWith("models")) {
            layoutNum++
        }

        if (fileList[x].name === "surfaces.ini") {
            goodFile = true;
        }
    }

    if (!goodFile) {
        notA("track");
        return
    }

    let $panel = $("#track-fail");
    $panel.hide();

    if (fileList[0].filepath.startsWith("tracks/")) {
        trackName = fileList[0].filepath.split("/")[1];
    } else {
        trackName = fileList[0].filepath.split("/")[0];
    }

    let tableDone = false;
    let $trackPanel = $("#track-info-panel");
    let $row = $("<div/>");
    let $title = $("<h3/>");

    $trackPanel.attr({'class': "card p-3 mt-2"});
    $title.text("Preview: " + trackName);
    $row.attr({'class': "card-deck"});

    $trackPanel.append($title);

    $trackPanel.append($row);

    for (let x = 0; x < filesToUploadLocal.length; x++) {
        if (filesToUploadLocal[x].filepath.includes("/ui/")) {

            if (filesToUploadLocal[x].name === "preview.png") {

                let layoutName = "";

                // For multiple layouts get the layout name and store in map
                if (layoutNum > 1) {
                    let fileListCorrected = filesToUploadLocal[x].filepath.replace('\\', '/');

                    let fileListSplit = fileListCorrected.split("/");

                    layoutName = fileListSplit[fileListSplit.length - 2];
                } else {
                    layoutName = "Default";
                }

                if (!layouts[layoutName]) {
                    layouts[layoutName] = Object.create(layout);
                }

                // Set preview to base64 encoded image
                let reader = new FileReader();

                reader.readAsDataURL(filesToUploadLocal[x]);

                reader.addEventListener("load", function () {
                    layouts[layoutName].preview = reader.result.toString();

                    let layoutInfo = layouts[layoutName];

                    $row.append(buildInfoPanel(layoutInfo.preview, "Layout: " + layoutName));
                });
            }

            if (filesToUploadLocal[x].name === "ui_track.json" && !tableDone) {
                tableDone = true;
                let reader = new FileReader();

                reader.readAsText(filesToUploadLocal[x]);

                reader.addEventListener("load", function () {
                    $trackPanel.append(buildHtmlTable([JSON.parse(reader.result.toString())]));
                });
            }
        }

    }

    let $uploadButton = $("#upload-button");
    $uploadButton.attr({'class': "d-inline"});

    if (filesToUploadLocal.length === 0) {
        $uploadButton.text("Sorry, the files you uploaded don't seem to be a compatible track!");
        $uploadButton.empty()
    } else {
        if (!$("#track-upload-button").length) {
            let $button = $("<button/>");
            $button.attr({
                'class': "btn btn-primary",
                'id': "track-upload-button"
            });
            $button.click(function (e) {
                e.preventDefault();

                submitFiles("/api/track/upload")
            });

            $button.text("Upload Track(s)");

            $uploadButton.append($button);
        }

        for (let x = 0; x < filesToUploadLocal.length; x++) {
            filesToUpload.push(filesToUploadLocal[x])
        }
    }
}

function notA(thing) {
    let $panel = $("#" + thing + "-fail");

    $panel.show();
    $panel.attr({'class': "alert alert-danger mt-2"});
    $panel.text("Sorry, looks like that wasn't a " + thing + "!")
}

function buildInfoPanel(img, info) {
    let $panel = $("<div/>");
    let $img = $("<img/>");
    let $cardBody = $("<div/>");
    let $cardText = $("<h5/>");

    $img.attr({'src': img});
    $img.attr({'alt': "Content Preview"});
    $img.attr({'class': "card-img-top"});

    $cardBody.attr({'class': "card-body"});

    $cardText.attr({'class': "card-title"});
    $cardText.text(info);

    $cardBody.append($cardText);

    $panel.append($img);
    $panel.append($cardBody);

    $panel.attr({'class': "card text-center mb-3"});

    return $panel;
}

// Builds a HTML table from JSON input.
function buildHtmlTable(json) {
    let $cardTable = $("<table/>");

    $cardTable.attr({'class': "table table-sm table-bordered"});
    $cardTable.attr({'id': "layout-table"});

    let columns = addAllColumnHeaders(json, $cardTable);

    for (let i = 0; i < json.length; i++) {
        let $row = $('<tr/>');
        for (let colIndex = 0; colIndex < columns.length; colIndex++) {
            let cellValue = json[i][columns[colIndex]] + "<br>";

            if (cellValue == null) {
                cellValue = "";
            }

            $row.append($('<td/>').html(cellValue));
        }
        $cardTable.append($row);
    }

    return $cardTable
}

// Adds a header row to the table and returns the set of columns.
function addAllColumnHeaders(json, table) {
    let columnSet = [];
    let headerTr$ = $('<tr/>');
    let header$ = $('<thead/>');

    header$.attr({'class': "table-secondary"});

    for (let i = 0; i < json.length; i++) {
        let rowHash = json[i];
        for (let key in rowHash) {
            if ($.inArray(key, columnSet) === -1) {
                if (key === "tags" || key === "run" || key === "url" || key === "torqueCurve" || key === "powerCurve") {
                    continue
                }

                columnSet.push(key);
                headerTr$.append($('<th/>').html(key));
            }
        }
    }

    header$.append(headerTr$);

    table.append(header$);

    return columnSet;
}


let championships = {
    init: function () {
        let $pointsTemplate = $document.find(".points-place").last().clone();

        $(".race-setup").each(function (index, elem) {
            // init totalNumPoints val to be equal to the number of .points-place's visible in the class.
            let $raceSetup = $(elem);
            let $savedNumPoints = $raceSetup.find(".totalNumPoints");
            $savedNumPoints.val($raceSetup.find(".points-place:visible").length);
        });

        $document.on("click", ".addEntrant", function (e) {
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
                let $newPoints = $pointsTemplate.clone();
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
        });

        championships.initClassSetup();
        championships.initSummerNote();
        championships.initDisplayOrder();
        championships.initConfigureSignUpForm();
        championships.initSignUpForm();
        championships.initACSRWatcher();
    },

    initACSRWatcher: function () {
        let $acsrSwitch = $("#ACSR");

        if ($acsrSwitch.length) {
            let state = $acsrSwitch.bootstrapSwitch('state');

            championships.setSwitchesForACSR(state);
        }

        $acsrSwitch.on('switchChange.bootstrapSwitch', function (event, state) {
            championships.setSwitchesForACSR(state);
        });
    },

    setSwitchesForACSR: function (state) {
        let $signUpFormSwitch = $("#ChampionshipSignUpFormEnabled");
        let $overridePasswordSwitch = $("#OverridePassword");

        if (state) {
            $signUpFormSwitch.bootstrapSwitch('state', true);
            $signUpFormSwitch.bootstrapSwitch('disabled', true);

            $overridePasswordSwitch.bootstrapSwitch('state', true);
            $overridePasswordSwitch.bootstrapSwitch('disabled', true);

            $overridePasswordSwitch.closest(".card-body").find("#ReplacementPasswordWrapper").hide();
        } else {
            $overridePasswordSwitch.closest(".card-body").find("#ReplacementPasswordWrapper").show();

            $signUpFormSwitch.bootstrapSwitch('disabled', false);
            $overridePasswordSwitch.bootstrapSwitch('disabled', false);
        }
    },

    initConfigureSignUpForm: function () {
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

        $document.on("click", ".btn-delete-question", function (e) {
            e.preventDefault();

            $(this).closest(".championship-signup-question").remove();
        })
    },

    initSignUpForm: function () {
        let $signUpForm = $("#championship-signup-form");

        if ($signUpForm.length < 1) {
            return;
        }

        function populateSkinsDropdown(car) {
            if (typeof availableCars === "undefined") {
                return;
            }

            let selected = $skinsDropdown.val();

            $skinsDropdown.empty();

            if (car in availableCars) {
                for (let skin of availableCars[car]) {
                    $skinsDropdown.append($("<option>", {
                        "val": skin,
                        "text": prettifyName(skin, true),
                        "selected": skin === selected,
                    }));
                }
            }
        }

        function showCarImage(car, skin) {
            let path = "/content/cars/" + encodeURIComponent(car) + "/skins/" + encodeURIComponent(skin) + "/preview.jpg";

            $.get(path)
                .done(function () {
                    $carPreviewImage.attr({"src": path, "alt": prettifyName(skin, false)})
                })
                .fail(function () {
                    path = "/static/img/no-preview-car.png";
                    $carPreviewImage.attr({"src": path, "alt": "Preview Image"})
                })
            ;
        }

        let $skinsDropdown = $signUpForm.find("#Skin");
        let $carsDropdown = $signUpForm.find("#Car");
        let $carPreviewImage = $signUpForm.find("#CarPreview");

        populateSkinsDropdown($carsDropdown.val());
        showCarImage($carsDropdown.val(), $skinsDropdown.val());

        $carsDropdown.on("change", function () {
            let $this = $(this);
            populateSkinsDropdown($this.val());
            showCarImage($this.val(), $skinsDropdown.val());
        });

        $skinsDropdown.on("change", function () {
            showCarImage($carsDropdown.val(), $skinsDropdown.val());
        });
    },

    $classTemplate: null,

    initClassSetup: function () {
        let $addClassButton = $document.find("#addClass");
        let $tmpl = $document.find("#class-template");
        championships.$classTemplate = $tmpl.clone();

        $tmpl.remove();

        $addClassButton.click(function (e) {
            e.preventDefault();

            let $cloned = championships.$classTemplate.clone().show();

            $(this).before($cloned);
            new RaceSetup($cloned);
            new CarSearch($cloned);

            let maxClients = $("#MaxClients").val();

            if ($(".entrant:visible").length >= maxClients) {
                $cloned.find(".entrant:visible").remove();
            }
        });

        $document.on("click", ".btn-delete-class", function (e) {
            e.preventDefault();
            $(this).closest(".race-setup").remove();
        });

        $document.on("input", ".entrant-team", function () {
            $(this).closest(".entrant").find(".points-transfer").show();
        });

        $(".Cars").each(function(index, element) {
            let $target = $(element);

            $target.closest(".race-setup").find("input[name='NumCars']").val($target.val().length);
        });

        $document.on("change", ".Cars", function(e) {
            let $target = $(e.currentTarget);

            $target.closest(".race-setup").find("input[name='NumCars']").val($target.val().length);
        });
    },

    $linkTemplate: null,

    initSummerNote: function () {
        if ($(".championship").length === 0 && $("#championship-form").length === 0) {
            return;
        }

        let $summerNote = $document.find("#summernote");
        let $ChampionshipInfoHolder = $document.find("#ChampionshipInfoHolder");

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
    },

    initDisplayOrder: function () {
        let $hideCompleted = $document.find("#hide-completed");
        let $hideNotCompleted = $document.find("#hide-not-completed");
        let $switchOrder = $document.find("#switch-order");


        let $championship = $document.find(".championship").first();

        $hideCompleted.click(function (e) {
            e.preventDefault();

            let $events = $document.find(".championship-event");

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

            let $events = $document.find(".championship-event");

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

            let $events = $document.find(".championship-event");

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
    },
};

function ordinalSuffix(i) {
    let j = i % 10,
        k = i % 100;
    if (j === 1 && k !== 11) {
        return i + "st";
    }
    if (j === 2 && k !== 12) {
        return i + "nd";
    }
    if (j === 3 && k !== 13) {
        return i + "rd";
    }

    return i + "th";
}
