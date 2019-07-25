import ClickEvent = JQuery.ClickEvent;

export class CarDetail {
    public constructor() {
        if (!$(".car-details").length) {
            return;
        }

        $(".car-image").on("click", CarDetail.onCarSkinClick);

        // make the car skins and hero-skin the same height
        CarDetail.fixCarImageHeights();
        $(window).on("resize", CarDetail.fixCarImageHeights);
        CarDetail.initSummerNote();
        CarDetail.initSkinUpload();
    }

    private static onCarSkinClick(e: ClickEvent) {
        const $currentTarget = $(e.currentTarget);

        $("#hero-skin").attr({
            "src": $currentTarget.attr("src"),
            "alt": $currentTarget.attr("alt"),
        });

        $("select[name='skin-delete']").val($currentTarget.data("skin"));
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

    private static initSkinUpload() {
        $("#input-folder-skin").on("change", () => {
            $("#upload-skin").show();
        });

        $("#skin-upload").on("submit", () => {
            const chooseFilesButton = $("#input-folder-skin").get(0) as HTMLInputElement;

            if (!chooseFilesButton.files) {
                return false;
            }

            // filter out files we're not interested in
            let list = new DataTransfer();

            for (let file of chooseFilesButton.files) {
                if (file.name === "livery.png" || file.name === "preview.jpg" || file.name === "ui_skin.json") {
                    list.items.add(file);
                }
            }

            chooseFilesButton.files = list.files;

            return true;
        });
    }
}