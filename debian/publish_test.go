package debian

import (
	"errors"
	"github.com/smira/aptly/database"
	. "launchpad.net/gocheck"
	"os"
	"path/filepath"
)

type NullSigner struct{}

func (n *NullSigner) SetKey(keyRef string) {

}

func (n *NullSigner) DetachedSign(source string, destination string) error {
	return nil
}

func (n *NullSigner) ClearSign(source string, destination string) error {
	return nil
}

type PublishedRepoSuite struct {
	PackageListMixinSuite
	repo              *PublishedRepo
	packageRepo       *Repository
	snapshot          *Snapshot
	db                database.Storage
	packageCollection *PackageCollection
}

var _ = Suite(&PublishedRepoSuite{})

func (s *PublishedRepoSuite) SetUpTest(c *C) {
	s.SetUpPackages()

	s.db, _ = database.OpenDB(c.MkDir())

	s.packageRepo = NewRepository(c.MkDir())

	repo, _ := NewRemoteRepo("yandex", "http://mirror.yandex.ru/debian/", "squeeze", []string{"main"}, []string{})
	repo.packageRefs = s.reflist

	s.snapshot, _ = NewSnapshotFromRepository("snap", repo)

	s.repo, _ = NewPublishedRepo("ppa", "squeeze", "main", nil, s.snapshot)

	s.packageCollection = NewPackageCollection(s.db)
	s.packageCollection.Update(s.p1)
	s.packageCollection.Update(s.p2)
	s.packageCollection.Update(s.p3)

	poolPath, _ := s.packageRepo.PoolPath(s.p1.Files[0].Filename, s.p1.Files[0].Checksums.MD5)
	err := os.MkdirAll(filepath.Dir(poolPath), 0755)
	f, err := os.Create(poolPath)
	c.Assert(err, IsNil)
	f.Close()
}

func (s *PublishedRepoSuite) TearDownTest(c *C) {
	s.db.Close()
}

func (s *PublishedRepoSuite) TestPrefixNormalization(c *C) {

	for _, t := range []struct {
		prefix        string
		expected      string
		errorExpected string
	}{
		{
			prefix:   "ppa",
			expected: "ppa",
		},
		{
			prefix:   "",
			expected: ".",
		},
		{
			prefix:   "/",
			expected: ".",
		},
		{
			prefix:   "//",
			expected: ".",
		},
		{
			prefix:   "//ppa/",
			expected: "ppa",
		},
		{
			prefix:   "ppa/..",
			expected: ".",
		},
		{
			prefix:   "ppa/ubuntu/",
			expected: "ppa/ubuntu",
		},
		{
			prefix:   "ppa/../ubuntu/",
			expected: "ubuntu",
		},
		{
			prefix:        "../ppa/",
			errorExpected: "invalid prefix .*",
		},
		{
			prefix:        "../ppa/../ppa/",
			errorExpected: "invalid prefix .*",
		},
		{
			prefix:        "ppa/dists",
			errorExpected: "invalid prefix .*",
		},
		{
			prefix:        "ppa/pool",
			errorExpected: "invalid prefix .*",
		},
	} {
		repo, err := NewPublishedRepo(t.prefix, "squeeze", "main", nil, s.snapshot)
		if t.errorExpected != "" {
			c.Check(err, ErrorMatches, t.errorExpected)
		} else {
			c.Check(repo.Prefix, Equals, t.expected)
		}
	}
}

func (s *PublishedRepoSuite) TestPublish(c *C) {
	err := s.repo.Publish(s.packageRepo, s.packageCollection, &NullSigner{})
	c.Assert(err, IsNil)

	c.Check(s.repo.Architectures, DeepEquals, []string{"i386"})

	rf, err := os.Open(filepath.Join(s.packageRepo.RootPath, "public/ppa/dists/squeeze/Release"))
	c.Assert(err, IsNil)

	cfr := NewControlFileReader(rf)
	st, err := cfr.ReadStanza()
	c.Assert(err, IsNil)

	c.Check(st["Origin"], Equals, "ppa squeeze")
	c.Check(st["Components"], Equals, "main")
	c.Check(st["Architectures"], Equals, "i386")

	pf, err := os.Open(filepath.Join(s.packageRepo.RootPath, "public/ppa/dists/squeeze/main/binary-i386/Packages"))
	c.Assert(err, IsNil)

	cfr = NewControlFileReader(pf)

	for i := 0; i < 3; i++ {
		st, err = cfr.ReadStanza()
		c.Assert(err, IsNil)

		c.Check(st["Filename"], Equals, "pool/main/a/alien-arena/alien-arena-common_7.40-2_i386.deb")
	}

	st, err = cfr.ReadStanza()
	c.Assert(err, IsNil)
	c.Assert(st, IsNil)

	_, err = os.Stat(filepath.Join(s.packageRepo.RootPath, "public/ppa/pool/main/a/alien-arena/alien-arena-common_7.40-2_i386.deb"))
	c.Assert(err, IsNil)
}

func (s *PublishedRepoSuite) TestString(c *C) {
	c.Check(s.repo.String(), Equals,
		"ppa/squeeze (main) publishes [snap]: Snapshot from mirror [yandex]: http://mirror.yandex.ru/debian/ squeeze")
	repo, _ := NewPublishedRepo("", "squeeze", "main", nil, s.snapshot)
	c.Check(repo.String(), Equals,
		"./squeeze (main) publishes [snap]: Snapshot from mirror [yandex]: http://mirror.yandex.ru/debian/ squeeze")
	repo, _ = NewPublishedRepo("", "squeeze", "main", []string{"i386", "amd64"}, s.snapshot)
	c.Check(repo.String(), Equals,
		"./squeeze (main) [i386, amd64] publishes [snap]: Snapshot from mirror [yandex]: http://mirror.yandex.ru/debian/ squeeze")
}

func (s *PublishedRepoSuite) TestKey(c *C) {
	c.Check(s.repo.Key(), DeepEquals, []byte("Uppa>>squeeze"))
}

func (s *PublishedRepoSuite) TestEncodeDecode(c *C) {
	encoded := s.repo.Encode()
	repo := &PublishedRepo{}
	err := repo.Decode(encoded)

	s.repo.snapshot = nil
	c.Assert(err, IsNil)
	c.Assert(repo, DeepEquals, s.repo)
}

type PublishedRepoCollectionSuite struct {
	PackageListMixinSuite
	db                  database.Storage
	snapshotCollection  *SnapshotCollection
	collection          *PublishedRepoCollection
	snap1, snap2        *Snapshot
	repo1, repo2, repo3 *PublishedRepo
}

var _ = Suite(&PublishedRepoCollectionSuite{})

func (s *PublishedRepoCollectionSuite) SetUpTest(c *C) {
	s.db, _ = database.OpenDB(c.MkDir())

	s.snapshotCollection = NewSnapshotCollection(s.db)

	s.snap1 = NewSnapshotFromPackageList("snap1", []*Snapshot{}, NewPackageList(), "desc1")
	s.snap2 = NewSnapshotFromPackageList("snap2", []*Snapshot{}, NewPackageList(), "desc2")

	s.snapshotCollection.Add(s.snap1)
	s.snapshotCollection.Add(s.snap2)

	s.repo1, _ = NewPublishedRepo("ppa", "anaconda", "main", []string{}, s.snap1)
	s.repo2, _ = NewPublishedRepo("", "anaconda", "main", []string{}, s.snap2)
	s.repo3, _ = NewPublishedRepo("ppa", "anaconda", "main", []string{}, s.snap2)

	s.collection = NewPublishedRepoCollection(s.db)
}

func (s *PublishedRepoCollectionSuite) TearDownTest(c *C) {
	s.db.Close()
}

func (s *PublishedRepoCollectionSuite) TestAddByName(c *C) {
	r, err := s.collection.ByPrefixDistribution("ppa", "anaconda")
	c.Assert(err, ErrorMatches, "*.not found")

	c.Assert(s.collection.Add(s.repo1), IsNil)
	c.Assert(s.collection.Add(s.repo1), ErrorMatches, ".*already exists")
	c.Assert(s.collection.CheckDuplicate(s.repo2), IsNil)
	c.Assert(s.collection.Add(s.repo2), IsNil)
	c.Assert(s.collection.Add(s.repo3), ErrorMatches, ".*already exists")
	c.Assert(s.collection.CheckDuplicate(s.repo3), Equals, s.repo1)

	r, err = s.collection.ByPrefixDistribution("ppa", "anaconda")
	c.Assert(err, IsNil)

	err = s.collection.LoadComplete(r, s.snapshotCollection)
	c.Assert(err, IsNil)
	c.Assert(r.String(), Equals, s.repo1.String())

	collection := NewPublishedRepoCollection(s.db)
	r, err = collection.ByPrefixDistribution("ppa", "anaconda")
	c.Assert(err, IsNil)

	err = s.collection.LoadComplete(r, s.snapshotCollection)
	c.Assert(err, IsNil)
	c.Assert(r.String(), Equals, s.repo1.String())
}

func (s *PublishedRepoCollectionSuite) TestByUUID(c *C) {
	r, err := s.collection.ByUUID(s.repo1.UUID)
	c.Assert(err, ErrorMatches, "*.not found")

	c.Assert(s.collection.Add(s.repo1), IsNil)

	r, err = s.collection.ByUUID(s.repo1.UUID)
	c.Assert(err, IsNil)

	err = s.collection.LoadComplete(r, s.snapshotCollection)
	c.Assert(err, IsNil)
	c.Assert(r.String(), Equals, s.repo1.String())
}

func (s *PublishedRepoCollectionSuite) TestUpdateLoadComplete(c *C) {
	c.Assert(s.collection.Update(s.repo1), IsNil)

	collection := NewPublishedRepoCollection(s.db)
	r, err := collection.ByPrefixDistribution("ppa", "anaconda")
	c.Assert(err, IsNil)
	c.Assert(r.snapshot, IsNil)
	c.Assert(s.collection.LoadComplete(r, s.snapshotCollection), IsNil)
	c.Assert(r.snapshot.UUID, Equals, s.repo1.snapshot.UUID)
}

func (s *PublishedRepoCollectionSuite) TestForEachAndLen(c *C) {
	s.collection.Add(s.repo1)

	count := 0
	err := s.collection.ForEach(func(*PublishedRepo) error {
		count++
		return nil
	})
	c.Assert(count, Equals, 1)
	c.Assert(err, IsNil)

	c.Check(s.collection.Len(), Equals, 1)

	e := errors.New("c")

	err = s.collection.ForEach(func(*PublishedRepo) error {
		return e
	})
	c.Assert(err, Equals, e)
}

type pathExistsChecker struct {
	*CheckerInfo
}

var PathExists = &pathExistsChecker{
	&CheckerInfo{Name: "PathExists", Params: []string{"path"}},
}

func (checker *pathExistsChecker) Check(params []interface{}, names []string) (result bool, error string) {
	_, err := os.Stat(params[0].(string))
	return err == nil, ""
}

type PublishedRepoRemoveSuite struct {
	PackageListMixinSuite
	db                         database.Storage
	snapshotCollection         *SnapshotCollection
	collection                 *PublishedRepoCollection
	packageRepo                *Repository
	snap1                      *Snapshot
	repo1, repo2, repo3, repo4 *PublishedRepo
}

var _ = Suite(&PublishedRepoRemoveSuite{})

func (s *PublishedRepoRemoveSuite) SetUpTest(c *C) {
	s.db, _ = database.OpenDB(c.MkDir())

	s.snapshotCollection = NewSnapshotCollection(s.db)

	s.snap1 = NewSnapshotFromPackageList("snap1", []*Snapshot{}, NewPackageList(), "desc1")

	s.snapshotCollection.Add(s.snap1)

	s.repo1, _ = NewPublishedRepo("ppa", "anaconda", "main", []string{}, s.snap1)
	s.repo2, _ = NewPublishedRepo("", "anaconda", "main", []string{}, s.snap1)
	s.repo3, _ = NewPublishedRepo("ppa", "meduza", "main", []string{}, s.snap1)
	s.repo4, _ = NewPublishedRepo("ppa", "osminog", "contrib", []string{}, s.snap1)

	s.collection = NewPublishedRepoCollection(s.db)
	s.collection.Add(s.repo1)
	s.collection.Add(s.repo2)
	s.collection.Add(s.repo3)
	s.collection.Add(s.repo4)

	s.packageRepo = NewRepository(c.MkDir())
	s.packageRepo.MkDir("ppa/dists/anaconda")
	s.packageRepo.MkDir("ppa/dists/meduza")
	s.packageRepo.MkDir("ppa/dists/osminog")
	s.packageRepo.MkDir("ppa/pool/main")
	s.packageRepo.MkDir("ppa/pool/contrib")
	s.packageRepo.MkDir("dists/anaconda")
	s.packageRepo.MkDir("pool/main")
}

func (s *PublishedRepoRemoveSuite) TearDownTest(c *C) {
	s.db.Close()
}

func (s *PublishedRepoRemoveSuite) TestRemoveFilesOnlyDist(c *C) {
	s.repo1.RemoveFiles(s.packageRepo, false, false)

	c.Check(filepath.Join(s.packageRepo.PublicPath(), "ppa/dists/anaconda"), Not(PathExists))
	c.Check(filepath.Join(s.packageRepo.PublicPath(), "ppa/dists/meduza"), PathExists)
	c.Check(filepath.Join(s.packageRepo.PublicPath(), "ppa/dists/osminog"), PathExists)
	c.Check(filepath.Join(s.packageRepo.PublicPath(), "ppa/pool/main"), PathExists)
	c.Check(filepath.Join(s.packageRepo.PublicPath(), "ppa/pool/contrib"), PathExists)
	c.Check(filepath.Join(s.packageRepo.PublicPath(), "dists/anaconda"), PathExists)
	c.Check(filepath.Join(s.packageRepo.PublicPath(), "pool/main"), PathExists)
}

func (s *PublishedRepoRemoveSuite) TestRemoveFilesWithPool(c *C) {
	s.repo1.RemoveFiles(s.packageRepo, false, true)

	c.Check(filepath.Join(s.packageRepo.PublicPath(), "ppa/dists/anaconda"), Not(PathExists))
	c.Check(filepath.Join(s.packageRepo.PublicPath(), "ppa/dists/meduza"), PathExists)
	c.Check(filepath.Join(s.packageRepo.PublicPath(), "ppa/dists/osminog"), PathExists)
	c.Check(filepath.Join(s.packageRepo.PublicPath(), "ppa/pool/main"), Not(PathExists))
	c.Check(filepath.Join(s.packageRepo.PublicPath(), "ppa/pool/contrib"), PathExists)
	c.Check(filepath.Join(s.packageRepo.PublicPath(), "dists/anaconda"), PathExists)
	c.Check(filepath.Join(s.packageRepo.PublicPath(), "pool/main"), PathExists)
}

func (s *PublishedRepoRemoveSuite) TestRemoveFilesWithPrefix(c *C) {
	s.repo1.RemoveFiles(s.packageRepo, true, true)

	c.Check(filepath.Join(s.packageRepo.PublicPath(), "ppa/dists/anaconda"), Not(PathExists))
	c.Check(filepath.Join(s.packageRepo.PublicPath(), "ppa/dists/meduza"), Not(PathExists))
	c.Check(filepath.Join(s.packageRepo.PublicPath(), "ppa/dists/osminog"), Not(PathExists))
	c.Check(filepath.Join(s.packageRepo.PublicPath(), "ppa/pool/main"), Not(PathExists))
	c.Check(filepath.Join(s.packageRepo.PublicPath(), "ppa/pool/contrib"), Not(PathExists))
	c.Check(filepath.Join(s.packageRepo.PublicPath(), "dists/anaconda"), PathExists)
	c.Check(filepath.Join(s.packageRepo.PublicPath(), "pool/main"), PathExists)
}

func (s *PublishedRepoRemoveSuite) TestRemoveFilesWithPrefixRoot(c *C) {
	s.repo2.RemoveFiles(s.packageRepo, true, true)

	c.Check(filepath.Join(s.packageRepo.PublicPath(), "ppa/dists/anaconda"), PathExists)
	c.Check(filepath.Join(s.packageRepo.PublicPath(), "ppa/dists/meduza"), PathExists)
	c.Check(filepath.Join(s.packageRepo.PublicPath(), "ppa/pool/main"), PathExists)
	c.Check(filepath.Join(s.packageRepo.PublicPath(), "ppa/pool/contrib"), PathExists)
	c.Check(filepath.Join(s.packageRepo.PublicPath(), "dists/anaconda"), Not(PathExists))
	c.Check(filepath.Join(s.packageRepo.PublicPath(), "pool/main"), Not(PathExists))
}

func (s *PublishedRepoRemoveSuite) TestRemoveRepo1and2(c *C) {
	err := s.collection.Remove(s.packageRepo, "ppa", "anaconda")
	c.Check(err, IsNil)

	_, err = s.collection.ByPrefixDistribution("ppa", "anaconda")
	c.Check(err, ErrorMatches, ".*not found")

	collection := NewPublishedRepoCollection(s.db)
	_, err = collection.ByPrefixDistribution("ppa", "anaconda")
	c.Check(err, ErrorMatches, ".*not found")

	c.Check(filepath.Join(s.packageRepo.PublicPath(), "ppa/dists/anaconda"), Not(PathExists))
	c.Check(filepath.Join(s.packageRepo.PublicPath(), "ppa/dists/meduza"), PathExists)
	c.Check(filepath.Join(s.packageRepo.PublicPath(), "ppa/dists/osminog"), PathExists)
	c.Check(filepath.Join(s.packageRepo.PublicPath(), "ppa/pool/main"), PathExists)
	c.Check(filepath.Join(s.packageRepo.PublicPath(), "ppa/pool/contrib"), PathExists)
	c.Check(filepath.Join(s.packageRepo.PublicPath(), "dists/anaconda"), PathExists)
	c.Check(filepath.Join(s.packageRepo.PublicPath(), "pool/main"), PathExists)

	err = s.collection.Remove(s.packageRepo, "ppa", "anaconda")
	c.Check(err, ErrorMatches, ".*not found")

	err = s.collection.Remove(s.packageRepo, "ppa", "meduza")
	c.Check(err, IsNil)

	c.Check(filepath.Join(s.packageRepo.PublicPath(), "ppa/dists/anaconda"), Not(PathExists))
	c.Check(filepath.Join(s.packageRepo.PublicPath(), "ppa/dists/meduza"), Not(PathExists))
	c.Check(filepath.Join(s.packageRepo.PublicPath(), "ppa/dists/osminog"), PathExists)
	c.Check(filepath.Join(s.packageRepo.PublicPath(), "ppa/pool/main"), Not(PathExists))
	c.Check(filepath.Join(s.packageRepo.PublicPath(), "ppa/pool/contrib"), PathExists)
	c.Check(filepath.Join(s.packageRepo.PublicPath(), "dists/anaconda"), PathExists)
	c.Check(filepath.Join(s.packageRepo.PublicPath(), "pool/main"), PathExists)
}

func (s *PublishedRepoRemoveSuite) TestRemoveRepo3(c *C) {
	err := s.collection.Remove(s.packageRepo, ".", "anaconda")
	c.Check(err, IsNil)

	_, err = s.collection.ByPrefixDistribution(".", "anaconda")
	c.Check(err, ErrorMatches, ".*not found")

	collection := NewPublishedRepoCollection(s.db)
	_, err = collection.ByPrefixDistribution(".", "anaconda")
	c.Check(err, ErrorMatches, ".*not found")

	c.Check(filepath.Join(s.packageRepo.PublicPath(), "ppa/dists/anaconda"), PathExists)
	c.Check(filepath.Join(s.packageRepo.PublicPath(), "ppa/dists/meduza"), PathExists)
	c.Check(filepath.Join(s.packageRepo.PublicPath(), "ppa/dists/osminog"), PathExists)
	c.Check(filepath.Join(s.packageRepo.PublicPath(), "ppa/pool/main"), PathExists)
	c.Check(filepath.Join(s.packageRepo.PublicPath(), "ppa/pool/contrib"), PathExists)
	c.Check(filepath.Join(s.packageRepo.PublicPath(), "dists/"), Not(PathExists))
	c.Check(filepath.Join(s.packageRepo.PublicPath(), "pool/"), Not(PathExists))
}
