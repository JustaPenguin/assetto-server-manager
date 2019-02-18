"use strict";

let $document;

// entry-point
$(document).ready(function () {
    console.log("initialising server manager javascript");

    $document = $(document);
    raceSetup.init();
    serverLogs.init();
    championships.init();

    // init bootstrap-switch
    $.fn.bootstrapSwitch.defaults.size = 'small';
    $.fn.bootstrapSwitch.defaults.animate = false;
    $.fn.bootstrapSwitch.defaults.onColor = "success";
    $document.find("input[type='checkbox']").bootstrapSwitch();

    $document.find('[data-toggle="tooltip"]').tooltip();

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

        $document.find("button[type='submit']").click(function() {
            canSubmit = true;
        });

        // ask the user before they close the webpage
        window.onbeforeunload = function() {
            if (canSubmit) {
                return;
            }

            return "Are you sure you want to navigate away? You'll lose unsaved changes to this setup if you do.";
        };
    }
});

const nameRegex = /^[A-Za-z]{0,5}[0-9]+/;

function prettifyName(name, acronyms) {
    let parts = name.split("_");

    if (parts[0] === "ks") {
        parts.shift();
    }

    for (let i = 0; i < parts.length; i++) {
        if ((acronyms && parts[i].length <= 3) || (acronyms && parts[i].match(nameRegex))) {
            parts[i] = parts[i].toUpperCase();
        } else {
            parts[i] = parts[i].split(' ')
                .map(w => w[0].toUpperCase() + w.substr(1).toLowerCase())
                .join(' ');
        }
    }

    return parts.join(" ")
}

let raceSetup = {
    // jQuery elements
    $trackDropdown: null,
    $trackLayoutDropdown: null,
    $trackLayoutDropdownParent: null,
    $carsDropdown: null,
    $tyresDropdown: null,
    $addWeatherButton: null,

    // the current layout as specified by the server
    currentLayout: "",

    // all available track layout options
    trackLayoutOpts: {},


    /**
     * init: entrypoint for raceSetup functions. looks for track + layout dropdowns and populates them.
     */
    init: function () {
        raceSetup.$carsDropdown = $document.find("#Cars");

        raceSetup.$trackDropdown = $document.find("#Track");
        raceSetup.$trackLayoutDropdown = $document.find("#TrackLayout");
        raceSetup.$trackLayoutDropdownParent = raceSetup.$trackLayoutDropdown.closest(".form-group");

        raceSetup.$addWeatherButton = $document.find("#addWeather");

        if (raceSetup.$carsDropdown) {
            raceSetup.$carsDropdown.multiSelect({
                selectableHeader: "<input type='search' class='form-control search-input' autocomplete='off' placeholder='search'>",
                selectionHeader: "<input type='search' class='form-control search-input' autocomplete='off' placeholder='search'>",
                afterInit: function (ms) {
                    let that = this,
                        $selectableSearch = that.$selectableUl.prev(),
                        $selectionSearch = that.$selectionUl.prev(),
                        selectableSearchString = '#' + that.$container.attr('id') + ' .ms-elem-selectable:not(.ms-selected)',
                        selectionSearchString = '#' + that.$container.attr('id') + ' .ms-elem-selection.ms-selected';

                    that.qs1 = $selectableSearch.quicksearch(selectableSearchString)
                        .on('keydown', function (e) {
                            if (e.which === 40) {
                                that.$selectableUl.focus();
                                return false;
                            }
                        });

                    that.qs2 = $selectionSearch.quicksearch(selectionSearchString)
                        .on('keydown', function (e) {
                            if (e.which === 40) {
                                that.$selectionUl.focus();
                                return false;
                            }
                        });
                },
                afterSelect: function () {
                    this.qs1.cache();
                    this.qs2.cache();
                },
                afterDeselect: function () {
                    this.qs1.cache();
                    this.qs2.cache();
                }
            });
            raceSetup.$tyresDropdown = $document.find("#LegalTyres");

            if (raceSetup.$tyresDropdown) {
                raceSetup.$tyresDropdown.multiSelect();

                raceSetup.$carsDropdown.change(raceSetup.populateTyreDropdown)
            }
        }

        raceSetup.$addWeatherButton.click(raceSetup.addWeather);

        $document.find(".weather-delete").click(function (e) {
            e.preventDefault();
            let $this = $(this);

            $this.closest(".weather").remove();

            // go through all .weather divs and update their numbers
            $document.find(".weather").each(function (index, elem) {
                $(elem).find(".weather-num").text(index);

            });
        });

        $document.find(".weather-graphics").change(raceSetup.updateWeatherGraphics);

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
            raceSetup.$trackLayoutDropdown.change(raceSetup.showTrackImage);

        }

        raceSetup.raceLaps();
        raceSetup.showEnabledSessions();
        raceSetup.showSolSettings();

        raceSetup.initEntrantsList();
        raceSetup.initSurfacePresets();
    },

    updateWeatherGraphics: function () {
        let $this = $(this);

        $this.closest(".weather").find(".weather-preview").attr({
            'src': '/content/weather/' + $this.val() + '/preview.jpg',
            'alt': $this.val(),
        });
    },

    /**
     * add weather elements to the form when the 'new weather' button is clicked
     */
    addWeather: function (e) {
        e.preventDefault();

        let $oldWeather = $document.find(".weather").last();

        let $newWeather = $oldWeather.clone(true, true);
        $newWeather.find(".weather-num").text($document.find(".weather").length);
        $newWeather.find(".weather-delete").show();

        $oldWeather.after($newWeather);
    },

    /**
     * when a session 'enabled' checkbox is modified, toggle the state of the session-details element
     */
    showEnabledSessions: function () {
        $(".session-enabler").each(function (index, elem) {
            $(elem).on('switchChange.bootstrapSwitch', function (event, state) {
                let $this = $(this);
                let $elem = $this.closest(".tab-pane").find(".session-details");
                let $panelLabel = $document.find("#" + $this.closest(".tab-pane").attr("aria-labelledby"));

                if (state) {
                    $elem.show();
                    $panelLabel.addClass("text-success")
                } else {
                    $elem.hide();
                    $panelLabel.removeClass("text-success")
                }
            });
        });
    },

    /**
     * when a Sol 'enabled' checkbox is modified, toggle the state of the sol-settings and not-sol-settings elements
     */
    showSolSettings: function () {
        $(".sol-enabler").each(function (index, elem) {
            raceSetup.showSolWeathers($(elem).val() === "on");

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

                raceSetup.showSolWeathers(state);
            });
        });
    },

    /**
     * hide non-sol weather if sol is enabled.
     *
     * @param state
     */
    showSolWeathers: function(state) {
        $document.find(".weather-graphics").each(function(graphicsIndex, graphicsElement) {
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
    },


    /**
     * populate the tyre dropdown for all currently selected cars.
     */
    populateTyreDropdown: function () {
        // quick race doesn't have tyre set up.
        if (typeof availableTyres === "undefined") {
            return
        }

        let cars = raceSetup.$carsDropdown.val();
        let allValidTyres = new Set();

        for (let index = 0; index < cars.length; index++) {
            let car = cars[index];
            let carTyres = availableTyres[car];

            for (let tyre in carTyres) {
                allValidTyres.add(tyre);

                if (raceSetup.$tyresDropdown.find("option[value='" + tyre + "']").length) {
                    continue; // this has already been added
                }

                raceSetup.$tyresDropdown.multiSelect('addOption', {
                    'value': tyre,
                    'text': carTyres[tyre] + " (" + tyre + ")",
                });

                raceSetup.$tyresDropdown.multiSelect('select', tyre);
            }
        }

        raceSetup.$tyresDropdown.find("option").each(function (index, elem) {
            let $elem = $(elem);

            if (!allValidTyres.has($elem.val())) {
                $elem.remove();

                raceSetup.$tyresDropdown.multiSelect('refresh');
            }
        });

    },

    /**
     * given a dropdown input which specifies 'laps'/'time', raceLaps will show the correct input element
     * and empty the unneeded one for either laps or race time.
     */
    raceLaps: function () {
        let $timeOrLaps = $document.find("#TimeOrLaps");
        let $raceLaps = $document.find("#RaceLaps");
        let $raceTime = $document.find("#RaceTime");
        let $extraLap = $document.find(".race-extra-lap");

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
    },

    /**
     * show track image shows the correct image for the track/layout combo
     */
    showTrackImage: function () {
        let track = raceSetup.$trackDropdown.val();
        let layout = raceSetup.$trackLayoutDropdown.val();

        let src = '/content/tracks/' + track + '/ui';

        if (layout) {
            src += '/' + layout;
        }

        // @TODO jpg
        src += '/preview.png';

        $document.find("#trackImage").attr({
            'src': src,
            'alt': track + ' ' + layout,
        })
    },

    /**
     * loadTrackLayouts: looks at the selected track and loads in the correct layouts for it into the
     * track layout dropdown
     */
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


        raceSetup.showTrackImage();
    },

    /**
     * buildTrackLayoutOption: builds an <option> containing track layout information
     * @param layout
     * @returns {HTMLElement}
     */
    buildTrackLayoutOption: function (layout) {
        let $opt = $("<option/>");
        $opt.attr({'value': layout});
        $opt.text(prettifyName(layout, true));

        if (layout === raceSetup.currentLayout) {
            $opt.prop("selected", true);
        }

        return $opt;
    },

    $entrantsDiv: null,
    $entrantTemplate: null,


    driverNames: [],

    autoCompleteDrivers: function () {
        let opts = {
            source: raceSetup.driverNames,
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

        $(document).on('keydown.autocomplete', ".entryListName", function () {
            $(this).autocomplete(opts);
        });
    },

    initEntrantsList: function () {
        raceSetup.$entrantsDiv = $document.find("#entrants");

        if (!raceSetup.$entrantsDiv.length) {
            return;
        }

        if (possibleEntrants) {
            for (let entrant of possibleEntrants) {
                raceSetup.driverNames.push(entrant.Name);
            }
        }

        function onEntryListCarChange() {
            let $this = $(this);
            let val = $this.val();

            populateEntryListSkins($this, val);
        }

        $document.find(".entryListCar").change(onEntryListCarChange);
        raceSetup.autoCompleteDrivers();

        let $tmpl = $document.find("#entrantTemplate");
        let $entrantTemplate = $tmpl.prop("id", "").clone(true, true);
        $tmpl.remove();

        function populateEntryListSkins($elem, val) {
            // populate skins
            let $skinsDropdown = $elem.closest(".entrant").find(".entryListSkin");
            let selected = $skinsDropdown.val();

            $skinsDropdown.empty();

            $("<option value='random_skin'>&lt;random skin&gt;</option>").appendTo($skinsDropdown);


            try {

                if (val in availableCars && availableCars[val] != null) {
                    for (let skin of availableCars[val]) {
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

        function deleteEntrant(e) {
            e.preventDefault();
            $(this).closest(".entrant").remove();
        }

        function populateEntryListCars() {
            // populate list of cars in entry list
            let cars = new Set(raceSetup.$carsDropdown.val());

            $document.find(".entryListCar").each(function (index, val) {
                let $val = $(val);
                let selected = $val.find("option:selected").val();

                if (!selected || !cars.has(selected)) {
                    selected = raceSetup.$carsDropdown.val()[0];
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

                populateEntryListSkins($val, selected);
            });
        }

        populateEntryListCars();
        raceSetup.$carsDropdown.change(populateEntryListCars);
        $document.find(".btn-delete-entrant").click(deleteEntrant);

        $document.find("#addEntrant").click(function (e) {
            e.preventDefault();

            let $numEntrantsField = $(this).parent().find("#numEntrantsToAdd");
            let numEntrantsToAdd = 1;

            if ($numEntrantsField.length > 0) {
                numEntrantsToAdd = $numEntrantsField.val();
            }

            for (let i = 0; i < numEntrantsToAdd; i++) {
                let $elem = $entrantTemplate.clone();
                $elem.find("input[type='checkbox']").bootstrapSwitch();
                $elem.insertBefore($(this).parent());
                $elem.find(".entryListCar").change(onEntryListCarChange);
                $elem.find(".btn-delete-entrant").click(deleteEntrant);
                populateEntryListCars();
                $elem.css("display", "block");
            }
        })

    },

    initSurfacePresets: function() {
        let $surfacePresetDropdown = $document.find("#SurfacePreset");

        if (!$surfacePresetDropdown.length) {
            return;
        }

        let $sessionStart = $document.find("#SessionStart");
        let $randomness = $document.find("#Randomness");
        let $sessionTransfer = $document.find("#SessionTransfer");
        let $lapGain = $document.find("#LapGain");

        $surfacePresetDropdown.change(function() {
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
    },
};

let logCharLimit = 500000;


let serverLogs = {
    init: function () {
        let $serverLog = $document.find("#server-logs");
        let $managerLog = $document.find("#manager-logs");

        let disableServerLogRefresh = false;
        let disableManagerLogRefresh = false;

        $serverLog.on("mousedown", function () {
            disableServerLogRefresh = true;
        });

        $serverLog.on("mouseup", function () {
            disableServerLogRefresh = false;
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

        if ($serverLog.length && $managerLog.length) {
            setInterval(function () {
                $.get("/api/logs", function (data) {
                    if (!window.getSelection().toString()) {
                        if (data.ServerLog.length > logCharLimit) {
                            data.ServerLog = data.ServerLog.slice(data.ServerLog.length - logCharLimit, data.ServerLog.length);
                        }

                        if (data.ManagerLog.length > logCharLimit) {
                            data.ManagerLog = data.ManagerLog.slice(data.ManagerLog.length - logCharLimit, data.ManagerLog.length);
                        }

                        if (isAtBottom($serverLog) && !disableServerLogRefresh) {

                            $serverLog.text(data.ServerLog);
                            $serverLog.scrollTop(1E10);
                        }

                        if (isAtBottom($managerLog) && !disableManagerLogRefresh) {
                            $managerLog.text(data.ManagerLog);
                            $managerLog.scrollTop(1E10);
                        }
                    }
                });
            }, 1000);
        }
    },
};

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

    // check for multiple weathers inside "weather" folder, if so call loop function for each weather
    if (fileList[0].filepath.startsWith("weather/") && !fileList[0].newPath) {
        let splitList = {};

        for (let y = 0; y < fileList.length; y++) {
            let splitPath = fileList[y].filepath.split("/");

            let weatherIdentifier = splitPath.slice(0, 2).join(":");

            fileList[y].newPath = splitPath.slice(1, splitPath.length - 1).join("/");

            if (!splitList[weatherIdentifier]) {
                splitList[weatherIdentifier] = []
            }

            splitList[weatherIdentifier].push(fileList[y]);
        }

        for (let weather in splitList) {
            handleWeatherFilesLoop(splitList[weather]);
        }
    } else {
        handleWeatherFilesLoop(fileList);
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

function handleCarFiles(fileList) {
    // Correct filepath
    for (let x = 0; x < fileList.length; x++) {
        if (!fileList[x].filepath) {
            fileList[x].filepath = fileList[x].webkitRelativePath;
        }
    }

    // check for multiple cars inside "cars" folder, if so recall this function for each car
    if (fileList[0].filepath.startsWith("cars/") && !fileList[0].newPath) {
        let splitList = {};

        for (let y = 0; y < fileList.length; y++) {
            let splitPath = fileList[y].filepath.split("/");

            let carIdentifier = splitPath.slice(0, 2).join(":");

            fileList[y].newPath = splitPath.slice(1, splitPath.length - 1).join("/");

            if (!splitList[carIdentifier]) {
                splitList[carIdentifier] = []
            }

            splitList[carIdentifier].push(fileList[y]);
        }

        for (let car in splitList) {
            handleCarFiles(splitList[car]);
        }
    } else {
        handleCarFilesLoop(fileList);
    }
}

function handleCarFilesLoop(fileList) {
    let filesToUploadLocal = [];
    let goodFile = false;

    for (let x = 0; x < fileList.length; x++) {
        // Find the files that the server is interested in
        if (fileList[x].name === "data.acd" || fileList[x].name === "ui_car.json"
            || fileList[x].name.startsWith("livery.") || fileList[x].name.startsWith("preview.")
            || fileList[x].name === "ui_skin.json") {

            filesToUploadLocal.push(fileList[x]);
            goodFile = true;
        }
    }

    if (!goodFile) {
        return
    }

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

    if (fileList[0].filepath.startsWith("tracks/") && !fileList[0].newPath) {
        let splitList = {};

        for (let y = 0; y < fileList.length; y++) {
            let splitPath = fileList[y].filepath.split("/");

            let trackIdentifier = splitPath.slice(0, 2).join(":");

            fileList[y].newPath = splitPath.slice(1, splitPath.length - 1).join("/");

            if (!splitList[trackIdentifier]) {
                splitList[trackIdentifier] = []
            }

            splitList[trackIdentifier].push(fileList[y]);
        }

        for (let track in splitList) {
            handleTrackFilesLoop(splitList[track]);
        }
    } else {
        handleTrackFilesLoop(fileList);
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
            (fileList[x].filepath.includes("/ui/"))) {

            filesToUploadLocal.push(fileList[x]);
            goodFile = true;
        }

        if (fileList[x].name.startsWith("models")) {
            layoutNum++
        }
    }

    if (!goodFile) {
        return
    }

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
    init: function() {
        let $pointsParent = $document.find("#points");

        if (!$pointsParent.length) {
            return;
        }

        let $pointsTemplate = $document.find(".points-place").last().clone();

        $document.find("#addEntrant").click(function() {
            let $points = $document.find(".points-place");
            let numEntrants = $document.find(".entrant").length;
            let numPoints = $points.length;

            for (let i = numPoints; i < numEntrants; i++) {
                // add points up to the numEntrants we have
                let $newPoints = $pointsTemplate.clone();
                $newPoints.find("label").text(ordinalSuffix(i+1) + " Place");

                let pointsVal = 0;

                // load the default points value for this position
                if (numPoints < defaultPoints.Places.length) {
                    pointsVal = defaultPoints.Places[i];
                }

                $newPoints.find("input").attr({"value": pointsVal});
                $pointsParent.append($newPoints);
            }
        });

        $document.on("click", ".btn-delete-entrant", function() {
            let numEntrants = $document.find(".entrant").length;
            let $points = $document.find(".points-place");
            let numPoints = $points.length;

            for (let i = numPoints; i >= numEntrants; i--) {
                // remove any extras we don't need
                $points.last().remove();
            }
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