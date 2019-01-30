"use strict";

let $document;

$(document).ready(function () {
    console.log("initialising server manager javascript");

    $document = $(document);

    tracks.init();
});

let tracks = {
    $trackDropdown: null,
    $trackLayoutDropdown: null,
    $trackLayoutDropdownParent: null,

    trackLayoutOpts: {},

    init: function () {
        tracks.$trackDropdown = $document.find("#Track");
        tracks.$trackLayoutDropdown = $document.find("#TrackLayout");
        tracks.$trackLayoutDropdownParent = tracks.$trackLayoutDropdown.closest(".form-group");

        if (tracks.$trackDropdown && tracks.$trackLayoutDropdown) {
            // build a map of track => available layouts
            tracks.$trackLayoutDropdown.find("option").each(function (index, opt) {
                let $optValSplit = $(opt).val().split(":");

                if (!tracks.trackLayoutOpts[$optValSplit[0]]) {
                    tracks.trackLayoutOpts[$optValSplit[0]] = [];
                }

                tracks.trackLayoutOpts[$optValSplit[0]].push($optValSplit[1]);
            });

            tracks.$trackLayoutDropdownParent.hide();
            tracks.loadTracks();

            tracks.$trackDropdown.change(tracks.loadTracks);
        }
    },

    loadTracks: function () {
        tracks.$trackLayoutDropdown.empty();

        let selectedTrack = tracks.$trackDropdown.find("option:selected").val();
        let availableLayouts = tracks.trackLayoutOpts[selectedTrack];

        if (availableLayouts) {
            for (let i = 0; i < availableLayouts.length; i++) {
                tracks.$trackLayoutDropdown.append(tracks.buildTrackLayoutOption(availableLayouts[i]));
            }

            tracks.$trackLayoutDropdownParent.show();
        } else {
            // add an option with an empty value
            tracks.$trackLayoutDropdown.append(tracks.buildTrackLayoutOption(""));
            tracks.$trackLayoutDropdownParent.hide();
        }
    },

    buildTrackLayoutOption: function (layout) {
        let $opt = $("<option/>");
        $opt.attr({'value': layout});
        $opt.text(layout);

        return $opt;
    },
};