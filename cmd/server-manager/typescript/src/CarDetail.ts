import ClickEvent = JQuery.ClickEvent;
import SubmitEvent = JQuery.SubmitEvent;

export class CarDetail {
    public constructor() {
        if ($(".car-details").length == 0) {
            return;
        }

        $(".car-image").on("click", CarDetail.onCarSkinClick);

        // make the car skins and hero-skin the same height
        CarDetail.fixCarImageHeights();
        $(window).on("resize", CarDetail.fixCarImageHeights);
    }

    private static onCarSkinClick(e: ClickEvent) {
        const $currentTarget = $(e.currentTarget);

        $("#hero-skin").attr({
            "src": $currentTarget.attr("src"),
            "alt": $currentTarget.attr("alt"),
        });
    }

    private static fixCarImageHeights() {
        $(".car-skins").height($("#hero-skin").height()!);
    }
}