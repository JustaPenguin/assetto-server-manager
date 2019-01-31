"use strict";

let $document;

// entry-point
$(document).ready(function () {
    console.log("initialising server manager javascript");

    $document = $(document);

    tracks.init();
});

let tracks = {
    // jQuery elements
    $trackDropdown: null,
    $trackLayoutDropdown: null,
    $trackLayoutDropdownParent: null,

    // the current layout as specified by the server
    currentLayout: "",

    // all available track layout options
    trackLayoutOpts: {},

    // init: entrypoint for tracks functions. looks for track + layout dropdowns and populates them.
    init: function () {
        tracks.$trackDropdown = $document.find("#Track");
        tracks.$trackLayoutDropdown = $document.find("#TrackLayout");
        tracks.$trackLayoutDropdownParent = tracks.$trackLayoutDropdown.closest(".form-group");

        // restrict loading track layouts to pages which have track dropdown and layout dropdown on them.
        if (tracks.$trackDropdown && tracks.$trackLayoutDropdown) {
            // build a map of track => available layouts
            tracks.$trackLayoutDropdown.find("option").each(function (index, opt) {
                let $optValSplit = $(opt).val().split(":");
                let trackName = $optValSplit[0];
                let trackLayout = $optValSplit[1];

                if (!tracks.trackLayoutOpts[trackName]) {
                    tracks.trackLayoutOpts[trackName] = [];
                }

                tracks.trackLayoutOpts[trackName].push(trackLayout);

                if ($optValSplit.length > 2) {
                    tracks.currentLayout = trackLayout;
                }
            });

            tracks.$trackLayoutDropdownParent.hide();
            tracks.loadTrackLayouts();

            tracks.$trackDropdown.change(tracks.loadTrackLayouts);
        }
    },

    // loadTrackLayouts: looks at the selected track and loads in the correct layouts for it into the
    // track layout dropdown
    loadTrackLayouts: function () {
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

    // buildTrackLayoutOption: builds an <option> containing track layout information
    buildTrackLayoutOption: function (layout) {
        let $opt = $("<option/>");
        $opt.attr({'value': layout});
        $opt.text(layout);

        if (layout === tracks.currentLayout) {
            $opt.prop("selected", true);
        }

        return $opt;
    },
};
