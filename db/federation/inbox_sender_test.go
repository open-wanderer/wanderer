package federation

import "testing"

func TestLocalDeliveryRequiresSenderToRemainLocalInStorage(t *testing.T) {
	f := setupAddressingTestApp(t)
	sender := f.actor(t, "sender", true)
	recipient := f.actor(t, "recipient", true)
	body := []byte(`{"type":"View","actor":"` + sender.GetString("iri") + `","object":"https://local.example/trail"}`)
	if handled, err := deliverLocalActivity(f.app, sender, recipient.GetString("inbox"), body); !handled || err != nil {
		t.Fatalf("stored local sender was not accepted: handled=%v err=%v", handled, err)
	}

	caller := sender.Clone()
	sender.Set("is_local", false)
	if err := f.app.Save(sender); err != nil {
		t.Fatal(err)
	}
	stored, err := f.app.FindRecordById("activitypub_actors", sender.Id)
	if err != nil {
		t.Fatal(err)
	}
	if stored.GetBool("is_local") || !caller.GetBool("is_local") || stored.GetString("iri") != caller.GetString("iri") || stored.GetString("private_key") != caller.GetString("private_key") {
		t.Fatal("fixture must change only the sender's stored local status")
	}
	if handled, err := deliverLocalActivity(f.app, caller, recipient.GetString("inbox"), body); !handled || err == nil {
		t.Fatalf("sender with a nonlocal stored record was accepted: handled=%v err=%v", handled, err)
	}
}
