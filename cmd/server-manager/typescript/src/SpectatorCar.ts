export class SpectatorCar {
    private $spectatorToggle: JQuery;

    public constructor() {
        this.$spectatorToggle = $(".spectator-toggle");

        if (!this.$spectatorToggle.length) {
            return;
        }

        this.$spectatorToggle.on('switchChange.bootstrapSwitch', () => { this.toggleSpectatorOptions() });
        this.toggleSpectatorOptions();
    }

    public toggleSpectatorOptions() {
        if (this.$spectatorToggle.bootstrapSwitch('state')) {
            $(".visible-spectator-enabled").show();
        } else {
            $(".visible-spectator-enabled").hide();
        }
    }
}
