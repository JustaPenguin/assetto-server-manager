import ClickEvent = JQuery.ClickEvent;
import {SummernoteWrapper} from "./forms/SummernoteWrapper";

export class TrackDetail {
    public constructor() {
        if (!$(".track-details").length) {
            return;
        }

        $(".track-image").on("click", TrackDetail.onTrackLayoutClick);

        TrackDetail.fixLayoutImageHeights();
        TrackDetail.initSummerNote();
        $(window).on("resize", TrackDetail.fixLayoutImageHeights);
    }

    private static onTrackLayoutClick(e: ClickEvent) {
        const $currentTarget = $(e.currentTarget);

        $("#hero-skin").attr({
            "src": $currentTarget.attr("src"),
            "alt": $currentTarget.attr("alt"),
        });

        $("select[name='skin-delete']").val($currentTarget.data("layout"));
    }

    private static fixLayoutImageHeights() {
        $(".track-layouts").height($("#hero-skin").height()!);
    }

    private static initSummerNote() {
        let $summerNote = $("#summernote");
        let $trackNotes = $("#TrackNotes");

        let html = "";

        if ($trackNotes.length > 0) {
            html = $trackNotes.html();
        }

        let wrapper = new SummernoteWrapper($summerNote, {
            placeholder: 'You can use this text input to attach notes to each track!',
            tabsize: 2,
            height: 200,
        }, html);

        wrapper.render();
    }
}
