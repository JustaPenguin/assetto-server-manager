import ClickEvent = JQuery.ClickEvent;

export class CarList {
    public constructor() {
        $(".delete-car").on("click", function(e: ClickEvent) {
            e.stopPropagation();

            return confirm("Are you sure that you want to permanently delete this car?");
        });

        $(".card-car").on("click", function() {
            window.location = $(this).data("href");
        });
    }
}