import ClickEvent = JQuery.ClickEvent;

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

        if ($trackNotes.length > 0) {
            $summerNote.summernote('code', $trackNotes.html());
        }

        $summerNote.summernote({
            placeholder: 'You can use this text input to attach notes to each track!',
            tabsize: 2,
            height: 200,
        });
    }
}