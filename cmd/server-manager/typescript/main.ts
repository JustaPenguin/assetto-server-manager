
import "jquery";
import "bootstrap";
import "bootstrap-switch";
import "randomcolor";
import "@fortawesome/fontawesome-free/js/all";
import {RaceSetup} from "./RaceSetup";
import {Championship} from "./Championship";

$(() => {
    console.log("initialising server manager javascript");

    new Championship();

    $(".race-setup").each(function (index, elem) {
        new RaceSetup($(elem));
    });

    $("#open-in-simres").each(function(index, elem) {
        let link = window.location.href.split("#")[0].replace("results", "results/download") + ".json";

        $(elem).attr('href', "http://simresults.net/remote?result=" + link);

        return false
    });

    /*
    serverLogs.init();
    liveTiming.init();
    liveMap.init();
    */

    // init bootstrap-switch
    $.fn.bootstrapSwitch.defaults.size = 'small';
    $.fn.bootstrapSwitch.defaults.animate = false;
    $.fn.bootstrapSwitch.defaults.onColor = "success";
    $("input[type='checkbox']").bootstrapSwitch();

    $('[data-toggle="tooltip"]').tooltip();
    $('[data-toggle="popover"]').popover();

    $(".row-link").click((e: JQuery.ClickEvent) => {
        window.location = $(e.currentTarget).data("href");
    });

    $(".results .driver-link").click((e: JQuery.ClickEvent) => {
        window.location = $(e.currentTarget).data("href");
        window.scrollBy(0, -100);
    });

    $('form').submit((e: JQuery.SubmitEvent) => {
        $(e.currentTarget).find('input[type="checkbox"]').each(function () {
            let $checkbox = $(e.currentTarget);
            if ($checkbox.is(':checked')) {
                $checkbox.attr('value', '1');
            } else {
                $checkbox.after().append($checkbox.clone().attr({type: 'hidden', value: 0}));
                $checkbox.prop('disabled', true);
            }
        })
    });

    if ($("form[data-safe-submit]").length > 0) {
        let canSubmit = false;

        $("button[type='submit']").click(function () {
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

    $("#CustomRaceScheduled").change((e: JQuery.ChangeEvent) => {
        if ($(e.currentTarget).val() && $("#CustomRaceScheduledTime").val()) {
            $("#start-race-button").hide();
            $("#save-race-button").val("schedule");
        } else {
            $("#start-race-button").show();
            $("#save-race-button").val("justSave");
        }
    });

    $("#CustomRaceScheduledTime").change((e: JQuery.ChangeEvent) => {
        if ($(e.currentTarget).val() && $("#CustomRaceScheduled").val()) {
            $("#start-race-button").hide();
            $("#save-race-button").val("schedule");
        } else {
            $("#start-race-button").show();
            $("#save-race-button").val("justSave");
        }
    });
});