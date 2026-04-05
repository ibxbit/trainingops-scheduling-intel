import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { ContentLibraryPage } from "../features/content/ContentLibraryPage";
import { useSessionStore } from "../state/session-store";

describe("Content upload edge validation", () => {
  beforeEach(() => {
    useSessionStore.setState({
      user: {
        userId: "u1",
        tenantId: "t1",
        roles: ["program_coordinator"],
        primaryRole: "program_coordinator",
      },
      isReady: true,
    });
  });

  it("blocks upload when file is missing", async () => {
    render(<ContentLibraryPage />);
    const uploadButton = screen.getByRole("button", {
      name: "Upload / Resume",
    });
    expect(uploadButton).toBeDisabled();
  });

  it("blocks invalid metadata ranges", async () => {
    render(<ContentLibraryPage />);
    const fileInput = document.querySelector(
      'input[type="file"]',
    ) as HTMLInputElement;
    const file = new File(["hello"], "hello.txt", { type: "text/plain" });
    await userEvent.upload(fileInput, file);
    const difficultyInput = screen.getByPlaceholderText("difficulty");
    await userEvent.clear(difficultyInput);
    await userEvent.type(difficultyInput, "6");
    await userEvent.click(
      screen.getByRole("button", { name: "Upload / Resume" }),
    );
    expect(screen.getByText("Difficulty must be between 1 and 5")).toBeTruthy();
  });
});
