declare var ShowUpgradePopup: boolean;

export class ChangelogPopup {
    public constructor() {
        if (!ShowUpgradePopup) {
            return;
        }

        const $changelogModal = $("#changelog-modal");

        $changelogModal.on('shown.bs.modal', () => {
            $.get("/accounts/dismiss-changelog");
        });

        $changelogModal.modal();
    }
}
