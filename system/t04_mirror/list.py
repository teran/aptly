from lib import BaseTest


class ListMirror1Test(BaseTest):
    """
    list mirrors: regular list
    """
    fixtureCmds = [
        "aptly mirror create mirror1 http://mirror.yandex.ru/debian/ wheezy",
        "aptly mirror create mirror2 http://mirror.yandex.ru/debian/ squeeze contrib",
        "aptly -architectures=i386 mirror create mirror3 http://mirror.yandex.ru/debian/ squeeze non-free",
    ]
    runCmd = "aptly mirror list"


class ListMirror2Test(BaseTest):
    """
    list mirrors: empty list
    """
    runCmd = "aptly mirror list"
