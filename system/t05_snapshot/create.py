from lib import BaseTest
import re


class CreateSnapshot1Test(BaseTest):
    """
    create snapshot: from mirror
    """
    fixtureDB = True
    runCmd = "aptly snapshot create snap1 from mirror wheezy-main"

    def check(self):
        def remove_created_at(s):
            return re.sub(r"Created At: [0-9:A-Za-z -]+\n", "", s)

        self.check_output()
        self.check_cmd_output("aptly snapshot show snap1", "snapshot_show", match_prepare=remove_created_at)


class CreateSnapshot2Test(BaseTest):
    """
    create snapshot: no mirror
    """
    fixtureDB = True
    runCmd = "aptly snapshot create snap1 from mirror no-such-mirror"
    expectedCode = 1


class CreateSnapshot3Test(BaseTest):
    """
    create snapshot: duplicate name
    """
    fixtureDB = True
    fixtureCmds = ["aptly snapshot create snap1 from mirror wheezy-main"]
    runCmd = "aptly snapshot create snap1 from mirror wheezy-contrib"
    expectedCode = 1
