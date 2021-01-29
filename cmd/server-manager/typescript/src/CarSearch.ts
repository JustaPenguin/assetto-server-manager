import ClickEvent = JQuery.ClickEvent;
import KeyPressEvent = JQuery.KeyPressEvent;

interface SearchResult {
    CarName: string;
    CarID: string;
    Class: string;
    // Tags: string[];
}

export class CarSearch {
    private $parent: JQuery<HTMLElement>;
    private $searchField: JQuery<HTMLInputElement>;
    private $searchButton: JQuery<HTMLButtonElement>;

    public constructor($parent: JQuery<HTMLElement>) {
        this.$parent = $parent;
        this.$searchField = $parent.find(".car-search") as JQuery<HTMLInputElement>;
        this.$searchButton = $parent.find(".car-search-btn") as JQuery<HTMLButtonElement>;

        if (!this.$searchField.length) {
            return;
        }

        this.$searchButton.on("click", (e: ClickEvent) => {
            e.preventDefault();
            this.doSearch();
        });

        this.$searchField.on("keypress", (e: KeyPressEvent) => {
            if (e.keyCode !== 13) { // enter key
                return;
            }

            e.preventDefault();
            this.doSearch();
        });
    }

    private doSearch() {
        const searchTerm = this.$searchField.val();
        const $carsSelect = this.$parent.find(".Cars");

        $.getJSON("/cars/search.json?q=" + encodeURIComponent(searchTerm as string), (data: SearchResult[]) => {
            // clear all unselected options
            $carsSelect.find("option:not(:selected)").remove();

            if (!data) {
                $carsSelect.multiSelect('refresh');
                return; // no search results
            }

            for (const car of data) {
                if ($carsSelect.find("option[value='" + car.CarID + "']").length) {
                    continue;
                }

                $carsSelect.append('<option value=' + car.CarID + " class=" + car.Class + ">" + car.CarName + "</option>")
            }

            $carsSelect.multiSelect('refresh');
        });
    }
}
