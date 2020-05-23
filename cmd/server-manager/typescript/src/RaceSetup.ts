import {initMultiSelect} from "./javascript/manager";
import {makeCarString, prettifyName} from "./utils";
import {EntryList} from "./EntryList";
import {CarSearch} from "./CarSearch";

declare var availableTyres, surfacePresets: any;

export class RaceSetup {
    // jQuery elements
    private $trackDropdown: JQuery<HTMLSelectElement>;
    private $trackLayoutDropdown: JQuery<HTMLSelectElement>;
    private $trackLayoutDropdownParent: JQuery<HTMLElement>;
    private readonly $carsDropdown: JQuery<HTMLSelectElement>;
    private readonly $tyresDropdown: JQuery | null = null;
    private $addWeatherButton: JQuery;

    // the current layout as specified by the server
    private currentLayout: string = "";

    // all available track layout options
    private readonly trackLayoutOpts: object;
    private readonly $parent: JQuery<HTMLElement>;
    private $document = $(document);


    // entryList for this RaceSetup
    private entryList: EntryList;
    private carSearch: CarSearch;

    public constructor($parent: JQuery<HTMLElement>) {
        this.$parent = $parent;
        this.trackLayoutOpts = {};
        this.carSearch = new CarSearch($parent);

        this.$carsDropdown = $parent.find(".Cars") as JQuery<HTMLSelectElement>;

        this.$trackDropdown = $parent.find("#Track") as JQuery<HTMLSelectElement>;
        this.$trackLayoutDropdown = $parent.find("#TrackLayout") as JQuery<HTMLSelectElement>;
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
                let $optValSplit = ($(opt).val() as string).split(":");
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

        this.entryList = new EntryList(this.$parent, this.$carsDropdown);

        this.raceLaps();
        this.showEnabledSessions();
        this.showSolSettings();
        this.toggleOverridePassword();

        this.initSunAngle();
        this.initSurfacePresets();
    }

    private updateWeatherGraphics(): void {
        let $this = $(this);

        $this.closest(".weather").find(".weather-preview").attr({
            'src': '/content/weather/' + $this.val() + '/preview.jpg',
            'alt': $this.val(),
        });
    }

    /**
     * add weather elements to the form when the 'new weather' button is clicked
     */
    private addWeather(e): void {
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
    private showEnabledSessions(): void {
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
    private toggleOverridePassword(): void {
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
    private showSolSettings(): void {
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
    private showSolWeathers(state): void {
        this.$parent.find(".weather-graphics").each(function (graphicsIndex, graphicsElement) {
            let $elem = $(graphicsElement);
            let $opts = $elem.find("option");
            let $selectedOpt = $elem.find("option:selected");

            if (state) {
                if (!/sol/i.test($selectedOpt.val() as string)) {
                    $elem.val("sol_01_CLear");
                }
            }

            for (let i = 0; i < $opts.length; i++) {
                let $opt = $($opts[i]);

                if (state && !/sol/i.test($opt.val() as string)) {
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
    private populateTyreDropdown(): void {
        // quick race doesn't have tyre set up.
        if (typeof availableTyres === "undefined" || !this.$carsDropdown.length || this.$tyresDropdown === null) {
            return
        }

        let cars = this.$carsDropdown.val() as Array<string>;

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

                that.$tyresDropdown!.multiSelect('refresh');
            }
        });
    }

    /**
     * given a dropdown input which specifies 'laps'/'time', raceLaps will show the correct input element
     * and empty the unneeded one for either laps or race time.
     */
    private raceLaps(): void {
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
    private showTrackDetails(): void {
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

        let $pitBoxes = this.$document.find("#track-pitboxes");
        let $maxClients = this.$document.find("#MaxClients");
        let $pitBoxesWarning = this.$document.find("#track-pitboxes-warning");

        let that = this;

        $.getJSON(jsonURL, function (trackInfo) {
            $pitBoxes.closest(".row").show();
            $pitBoxes.text(trackInfo.pitboxes);

            let $entrantIDs = that.$document.find(".entrant-id");
            $entrantIDs.attr("max", (trackInfo.pitboxes - 1));

            let entrants = that.$document.find(".entrant").length;

            if (entrants > trackInfo.pitboxes) {
                $pitBoxesWarning.show()
            } else {
                $pitBoxesWarning.hide()
            }

            let overrideAmount = $maxClients.data('value-override');

            if ((overrideAmount && trackInfo.pitboxes <= overrideAmount) || !overrideAmount) {
                $maxClients.attr("max", trackInfo.pitboxes);

                if (parseInt($maxClients.val() as string) > trackInfo.pitboxes) {
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
    private loadTrackLayouts(): void {
        this.$trackLayoutDropdown.empty();

        let selectedTrack = this.$trackDropdown.find("option:selected").val() as string;
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
    private buildTrackLayoutOption(layout): JQuery<HTMLElement> {
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


    private initSunAngle() {
        let $timeOfDay = this.$parent.find("#TimeOfDay");
        let $sunAngle = this.$parent.find("#SunAngle");

        function updateTime() {
            let angle = $sunAngle.val();
            let time = getTime(angle);

            $timeOfDay.val(time.getHours() + ":" + getFullMinutes(time));
        }

        updateTime();

        $timeOfDay.change(function () {
            let split = ($(this).val() as string).split(':');

            if (split.length < 2) {
                return;
            }

            $sunAngle.val(getSunAngle(split[0], split[1]));
        });

        $sunAngle.change(updateTime);
    }

    private initSurfacePresets() {
        let $surfacePresetDropdown = this.$parent.find("#SurfacePreset");

        if (!$surfacePresetDropdown.length) {
            return;
        }

        let $sessionStart = this.$parent.find("#SessionStart");
        let $randomness = this.$parent.find("#Randomness");
        let $sessionTransfer = this.$parent.find("#SessionTransfer");
        let $lapGain = this.$parent.find("#LapGain");

        $surfacePresetDropdown.change(function () {
            let val = $surfacePresetDropdown.val() as string;

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
    let diff = time.getTime() - baseLine.getTime();

    return Math.round(((diff / 60000) / 60) * 16);
}

function getFullMinutes(date: Date): string {
    if (date.getMinutes() < 10) {
        return '0' + date.getMinutes();
    }
    return '' + date.getMinutes();
}
