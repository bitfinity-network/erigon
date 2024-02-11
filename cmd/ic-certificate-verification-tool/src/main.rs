use candid::Principal;
use did::{certified_result::CertifiedResult, Block, H256};
use ic_cbor::{CertificateToCbor, HashTreeToCbor};
use ic_certificate_verification::VerifyCertificate;
use ic_certification::{Certificate, HashTree, LookupResult};

fn main() -> Result<(), anyhow::Error> {
    let mut args = std::env::args();
    if args.len() != 4 {
        anyhow::bail!("Invalid arguments number. Usage: ic-certificate-varification-tool <CERTIFIED_BLOCK> <CANISTER_ID> <ROOT_KEY>");
    }

    args.next().expect("should be 4 arguments");
    let certified_response: CertifiedResult<Block<H256>> =
        serde_json::from_str(&args.next().expect("should be 4 arguments"))?;
    let canister_id = Principal::from_text(&args.next().expect("should be 4 arguments"))?;
    let root_key = hex::decode(&args.next().expect("should be 4 arguments"))?;

    let certificate = Certificate::from_cbor(&certified_response.certificate).unwrap();
    certificate.verify(canister_id.as_ref(), &root_key)?;

    let tree = HashTree::from_cbor(&certified_response.witness).unwrap();

    if !validate_tree(canister_id.as_slice(), &certificate, &tree) {
        anyhow::bail!("Signature verification failed");
    }

    return Ok(());
}

fn validate_tree(canister_id: &[u8], certificate: &Certificate, tree: &HashTree) -> bool {
    let certified_data_path = [
        "canister".as_bytes(),
        canister_id,
        "certified_data".as_bytes(),
    ];

    let witness = match certificate.tree.lookup_path(&certified_data_path) {
        LookupResult::Found(witness) => witness,
        _ => {
            return false;
        }
    };

    let digest = tree.digest();
    if witness != digest {
        return false;
    }

    true
}
