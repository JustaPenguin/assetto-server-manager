"use strict";

let $document;

// entry-point
$(document).ready(function () {
    console.log("initialising server manager javascript");

    $document = $(document);

    // init bootstrap-switch
    $.fn.bootstrapSwitch.defaults.size = 'small';
    $.fn.bootstrapSwitch.defaults.animate = false;
    $.fn.bootstrapSwitch.defaults.onColor = "success";
    $document.find("input[type='checkbox']").bootstrapSwitch();

    raceSetup.init();
    serverLogs.init();
});

let raceSetup = {
    // jQuery elements
    $trackDropdown: null,
    $trackLayoutDropdown: null,
    $trackLayoutDropdownParent: null,
    $carsDropdown: null,
    $tyresDropdown: null,

    // the current layout as specified by the server
    currentLayout: "",

    // all available track layout options
    trackLayoutOpts: {},

    // current tyres present in tyres multiselect.
    carTyres: {},

    // init: entrypoint for raceSetup functions. looks for track + layout dropdowns and populates them.
    init: function () {
        raceSetup.$carsDropdown = $document.find("#Cars");

        raceSetup.$trackDropdown = $document.find("#Track");
        raceSetup.$trackLayoutDropdown = $document.find("#TrackLayout");
        raceSetup.$trackLayoutDropdownParent = raceSetup.$trackLayoutDropdown.closest(".form-group");

        if (raceSetup.$carsDropdown) {
            raceSetup.$carsDropdown.multiSelect();
            raceSetup.$tyresDropdown = $document.find("#LegalTyres");

            if (raceSetup.$tyresDropdown) {
                raceSetup.$tyresDropdown.multiSelect();

                raceSetup.$carsDropdown.change(function(evt) {
                    let cars = raceSetup.$carsDropdown.val();

                    for (let index = 0; index < cars.length; index++) {
                        let car = cars[index];
                        let carTyres = availableTyres[car];

                        for (let tyre in carTyres) {
                            if (raceSetup.carTyres[tyre]) {
                                continue; // this has already been added
                            }

                            let $opt = $("<option/>");

                            $opt.attr({'value': tyre});
                            $opt.text(carTyres[tyre] + " (" + tyre + ")");

                            raceSetup.$tyresDropdown.append($opt);

                            raceSetup.carTyres[tyre] = true;
                        }
                    }

                    raceSetup.$tyresDropdown.multiSelect('refresh');
                })
            }
        }

        // restrict loading track layouts to pages which have track dropdown and layout dropdown on them.
        if (raceSetup.$trackDropdown.length && raceSetup.$trackLayoutDropdown.length) {
            // build a map of track => available layouts
            raceSetup.$trackLayoutDropdown.find("option").each(function (index, opt) {
                let $optValSplit = $(opt).val().split(":");
                let trackName = $optValSplit[0];
                let trackLayout = $optValSplit[1];

                if (!raceSetup.trackLayoutOpts[trackName]) {
                    raceSetup.trackLayoutOpts[trackName] = [];
                }

                raceSetup.trackLayoutOpts[trackName].push(trackLayout);

                if ($optValSplit.length > 2) {
                    raceSetup.currentLayout = trackLayout;
                }
            });

            raceSetup.$trackLayoutDropdownParent.hide();
            raceSetup.loadTrackLayouts();

            raceSetup.$trackDropdown.change(raceSetup.loadTrackLayouts);
        }

        raceSetup.raceLaps();
        raceSetup.showEnabledSessions();
    },

    showEnabledSessions: function() {
        $(".session-enabler").each(function(index, elem) {
            $(elem).on('switchChange.bootstrapSwitch',function(event, state) {
                let $this = $(this);
                let $elem = $this.closest(".tab-pane").find(".session-details");

                if (state) {
                    $elem.show();
                } else {
                    $elem.hide();
                }
            });
        });
    },

    raceLaps: function() {
        let $timeOrLaps = $document.find("#TimeOrLaps");
        let $raceLaps = $document.find("#RaceLaps");
        let $raceTime = $document.find("#RaceTime");

        if ($timeOrLaps.length) {
            $timeOrLaps.change(function() {
                let selected = $timeOrLaps.find("option:selected").val();

                if (selected === "Time") {
                    $raceLaps.find("input").val(0);
                    $raceTime.find("input").val(15);
                    $raceLaps.hide();
                    $raceTime.show();
                } else {
                    $raceTime.find("input").val(0);
                    $raceLaps.find("input").val(10);
                    $raceLaps.show();
                    $raceTime.hide();
                }
            });
        }
    },

    // loadTrackLayouts: looks at the selected track and loads in the correct layouts for it into the
    // track layout dropdown
    loadTrackLayouts: function () {
        raceSetup.$trackLayoutDropdown.empty();

        let selectedTrack = raceSetup.$trackDropdown.find("option:selected").val();
        let availableLayouts = raceSetup.trackLayoutOpts[selectedTrack];

        if (availableLayouts) {
            for (let i = 0; i < availableLayouts.length; i++) {
                raceSetup.$trackLayoutDropdown.append(raceSetup.buildTrackLayoutOption(availableLayouts[i]));
            }

            raceSetup.$trackLayoutDropdownParent.show();
        } else {
            // add an option with an empty value
            raceSetup.$trackLayoutDropdown.append(raceSetup.buildTrackLayoutOption(""));
            raceSetup.$trackLayoutDropdownParent.hide();
        }
    },

    // buildTrackLayoutOption: builds an <option> containing track layout information
    buildTrackLayoutOption: function (layout) {
        let $opt = $("<option/>");
        $opt.attr({'value': layout});
        $opt.text(layout);

        if (layout === raceSetup.currentLayout) {
            $opt.prop("selected", true);
        }

        return $opt;
    },
};


let serverLogs = {
    init: function() {
        let $serverLog = $document.find("#server-logs");
        let $managerLog = $document.find("#manager-logs");

        if ($serverLog.length && $managerLog.length) {

            setInterval(function() {
                $.get("/api/logs", function (data) {
                    $serverLog.text(data.ServerLog);
                    $managerLog.text(data.ManagerLog);
                });
            }, 1000);
        }
    },
};