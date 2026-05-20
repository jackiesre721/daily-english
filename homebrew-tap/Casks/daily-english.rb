cask "daily-english" do
  version "1.0.0"
  sha256 "UPDATE_AFTER_FIRST_RELEASE"

  url "https://github.com/USER/daily-english/releases/download/v#{version}/Daily-English-#{version}.dmg"
  name "Daily English"
  desc "English reading and vocabulary learning companion"
  homepage "https://github.com/USER/daily-english"

  depends_on macos: ">= :high_sierra"

  app "Daily English.app"

  zap trash: [
    "~/Library/Application Support/DailyEnglish",
  ]
end
