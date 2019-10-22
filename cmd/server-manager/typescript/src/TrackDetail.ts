import ClickEvent = JQuery.ClickEvent;

export class TrackDetail {
    public constructor() {
        if (!$(".track-details").length) {
            return;
        }

        $(".track-image").on("click", TrackDetail.onTrackLayoutClick);

        TrackDetail.fixLayoutImageHeights();
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
}