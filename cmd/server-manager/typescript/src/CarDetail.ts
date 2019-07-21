import ClickEvent = JQuery.ClickEvent;

export class CarDetail {
    public constructor() {
        if ($(".car-details").length == 0) {
            return;
        }

        $(".car-image").on("click", CarDetail.onCarSkinClick);

        // make the car skins and hero-skin the same height
        CarDetail.fixCarImageHeights();
        $(window).on("resize", CarDetail.fixCarImageHeights);
        CarDetail.initSummerNote();
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

    private static initSummerNote() {
        let $summerNote = $("#summernote");
        let $carNotes = $("#CarNotes");

        if ($carNotes.length > 0) {
            $summerNote.summernote('code', $carNotes.html());
        }

        $summerNote.summernote({
            placeholder: 'You can use this text input to attach notes to each car!',
            tabsize: 2,
            height: 200,
        });
    }
}